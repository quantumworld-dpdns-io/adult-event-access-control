using HTTP
using JSON
using DataFrames
using Dates
using Statistics
using CSV
using .DB
using .FraudDetection

const PORT = parse(Int, get(ENV, "PORT", "8090"))

const RATE_LIMIT_MAX = parse(Int, get(ENV, "RATE_LIMIT_MAX", "100"))
const RATE_LIMIT_WINDOW = 60.0
const request_log = Dict{String, Tuple{Int, Float64}}()

function check_rate_limit(req::HTTP.Request)
    ip = get(req.headers, "X-Forwarded-For", "127.0.0.1")
    now_t = time()
    if haskey(request_log, ip)
        count, window_start = request_log[ip]
        if now_t - window_start > RATE_LIMIT_WINDOW
            request_log[ip] = (1, now_t)
            return true
        elseif count >= RATE_LIMIT_MAX
            return false
        else
            request_log[ip] = (count + 1, window_start)
            return true
        end
    else
        request_log[ip] = (1, now_t)
        return true
    end
end

function json_response(status, body)
    return HTTP.Response(status, ["Content-Type" => "application/json"], JSON.json(body))
end

function extract_event_id(target)
    parts = split(target, "/")
    if length(parts) >= 5
        return parts[5]
    end
    return nothing
end

function fetch_tickets(conn)
    if conn === nothing
        return DataFrame(
            event_id = String[],
            status = String[],
            checked_in_at = Union{Missing,DateTime}[],
            issued_at = Union{Missing,DateTime}[],
            event_title = String[],
            start_time = Union{Missing,DateTime}[]
        )
    end
    result = execute(conn, """
        SELECT t.id, t.event_id, t.status, t.checked_in_at, t.issued_at,
               e.title AS event_title, e.start_time
        FROM tickets t
        JOIN events e ON e.id = t.event_id
        ORDER BY t.issued_at DESC
    """)
    return DataFrame(result)
end

function fetch_tickets_for_event(conn, event_id)
    if conn === nothing
        return DataFrame(
            id = String[],
            status = String[],
            checked_in_at = Union{Missing,DateTime}[],
            issued_at = Union{Missing,DateTime}[]
        )
    end
    result = execute(conn, """
        SELECT t.id, t.status, t.checked_in_at, t.issued_at
        FROM tickets t
        WHERE t.event_id = \$1
        ORDER BY t.issued_at DESC
    """, [event_id])
    return DataFrame(result)
end

function fetch_transfers(conn)
    if conn === nothing
        return DataFrame(
            ticket_id = String[],
            from_user_id = String[],
            to_user_id = String[],
            created_at = DateTime[],
            event_id = String[]
        )
    end
    result = execute(conn, """
        SELECT tl.*, t.event_id
        FROM transfer_log tl
        JOIN tickets t ON t.id = tl.ticket_id
        ORDER BY tl.created_at DESC
    """)
    return DataFrame(result)
end

function fetch_transfers_for_event(conn, event_id)
    if conn === nothing
        return DataFrame(
            ticket_id = String[],
            from_user_id = String[],
            to_user_id = String[],
            created_at = DateTime[]
        )
    end
    result = execute(conn, """
        SELECT tl.*
        FROM transfer_log tl
        JOIN tickets t ON t.id = tl.ticket_id
        WHERE t.event_id = \$1
        ORDER BY tl.created_at DESC
    """, [event_id])
    return DataFrame(result)
end

function detect_anomalies(tickets_df, transfers_df)
    results = Dict[]
    if nrow(transfers_df) > 0 && nrow(tickets_df) > 0
        if :event_id in names(transfers_df) && :event_id in names(tickets_df)
            event_transfer_counts = combine(groupby(transfers_df, :event_id), nrow => :transfers)
            event_ticket_counts = combine(groupby(tickets_df, :event_id), nrow => :tickets)
            merged = leftjoin(event_ticket_counts, event_transfer_counts, on = :event_id)
            merged[!, :transfers] = coalesce.(merged[!, :transfers], 0)
            merged[!, :transfer_ratio] = merged[!, :transfers] ./ max.(merged[!, :tickets], 1)
            mean_ratio = mean(merged[!, :transfer_ratio])
            std_ratio = std(merged[!, :transfer_ratio])
            for row in eachrow(merged)
                if std_ratio > 0 && row.transfer_ratio > mean_ratio + 2 * std_ratio
                    push!(results, Dict(
                        "type" => "high_transfer_ratio",
                        "event_id" => row.event_id,
                        "score" => round((row.transfer_ratio - mean_ratio) / std_ratio, digits=2),
                        "transfer_ratio" => round(row.transfer_ratio, digits=3),
                        "severity" => "high"
                    ))
                end
            end
        end
    end
    if nrow(tickets_df) > 0
        if :status in names(tickets_df) && :checked_in_at in names(tickets_df)
            checked_in = filter(:status => s -> s == "checked_in", tickets_df)
            if nrow(checked_in) > 1
                times = sort(checked_in[!, :checked_in_at])
                gaps = diff([Float64(t) / 1000 for t in times])
                if length(gaps) > 0
                    mean_gap = mean(gaps)
                    std_gap = std(gaps)
                    for (i, gap) in enumerate(gaps)
                        if std_gap > 0 && gap < mean_gap - 2 * std_gap && gap < 1.0
                            push!(results, Dict(
                                "type" => "rapid_checkin",
                                "check_in_index" => i + 1,
                                "gap_seconds" => round(gap, digits=2),
                                "severity" => "medium"
                            ))
                        end
                    end
                end
            end
        end
    end
    return results
end

function sybil_score(conn, user_id::String)
    if conn === nothing
        return Dict("sybil_score" => 0.0, "risk_level" => "low")
    end
    result = execute(conn, """
        SELECT COUNT(DISTINCT event_id) AS event_count,
               COUNT(*) AS ticket_count,
               BOOL_OR(wp.nullifier_hash IS NOT NULL) AS has_world_id
        FROM tickets t
        LEFT JOIN world_id_proofs wp ON wp.user_id = t.owner_id
        WHERE t.owner_id = \$1
    """, [user_id])
    df = DataFrame(result)
    if nrow(df) == 0
        return Dict("sybil_score" => 1.0, "risk_level" => "high")
    end
    row = first(df)
    has_world_id = coalesce(get(row, :has_world_id, false), false)
    event_count = coalesce(get(row, :event_count, 0), 0)
    score = has_world_id ? 0.0 : min(1.0, 0.3 + 0.1 * (1.0 / max(event_count, 1)))
    risk = score < 0.2 ? "low" : score < 0.5 ? "medium" : "high"
    return Dict(
        "sybil_score" => round(score, digits=3),
        "risk_level" => risk,
        "has_world_id" => has_world_id,
        "events_attended" => event_count
    )
end

function route(req::HTTP.Request)
    if !check_rate_limit(req)
        return json_response(429, Dict("error" => "rate limit exceeded"))
    end

    target = string(req.target)
    method = req.method

    if method == "GET" && target == "/health"
        return json_response(200, Dict("status" => "ok", "service" => "aev-analytics"))
    end

    if method == "GET" && target == "/api/analytics/overview"
        conn = DB.get_connection()
        tickets = fetch_tickets(conn)
        transfers = fetch_transfers(conn)
        overview = Dict(
            "total_tickets" => nrow(tickets),
            "total_transfers" => nrow(transfers),
            "transfer_rate" => nrow(tickets) > 0 ? round(nrow(transfers) / nrow(tickets), digits=4) : 0.0,
            "checked_in" => nrow(tickets) > 0 ? nrow(filter(:status => s -> s == "checked_in", tickets)) : 0,
            "anomalies" => detect_anomalies(tickets, transfers)
        )
        DB.close_connection(conn)
        return json_response(200, overview)
    end

    if method == "GET" && target == "/api/analytics/anomalies"
        conn = DB.get_connection()
        tickets = fetch_tickets(conn)
        transfers = fetch_transfers(conn)
        DB.close_connection(conn)
        return json_response(200, Dict("anomalies" => detect_anomalies(tickets, transfers)))
    end

    if method == "GET" && startswith(target, "/api/analytics/sybil/") && length(split(target, "/")) >= 5
        user_id = split(target, "/")[end]
        if isempty(user_id)
            return json_response(400, Dict("error" => "missing user_id"))
        end
        conn = DB.get_connection()
        result = sybil_score(conn, user_id)
        DB.close_connection(conn)
        return json_response(200, result)
    end

    m_event_detailed = match(r"^/api/analytics/event/([^/]+)/detailed$", target)
    if method == "GET" && m_event_detailed !== nothing
        event_id = m_event_detailed.captures[1]
        conn = DB.get_connection()
        tickets = fetch_tickets_for_event(conn, event_id)
        transfers = fetch_transfers_for_event(conn, event_id)
        DB.close_connection(conn)
        checked_in = nrow(tickets) > 0 ? nrow(filter(:status => s -> s == "checked_in", tickets)) : 0
        return json_response(200, Dict(
            "event_id" => event_id,
            "total_tickets" => nrow(tickets),
            "checked_in" => checked_in,
            "total_transfers" => nrow(transfers),
            "anomalies" => detect_anomalies(tickets, transfers)
        ))
    end

    if method == "GET" && target == "/api/analytics/export"
        conn = DB.get_connection()
        tickets = fetch_tickets(conn)
        transfers = fetch_transfers(conn)
        DB.close_connection(conn)
        tickets_csv = ""
        transfers_csv = ""
        if nrow(tickets) > 0
            tickets_csv = CSV.write(IOBuffer(), tickets) |> String
        end
        if nrow(transfers) > 0
            transfers_csv = CSV.write(IOBuffer(), transfers) |> String
        end
        body = "--- TICKETS ---\n$tickets_csv\n--- TRANSFERS ---\n$transfers_csv"
        return HTTP.Response(200, ["Content-Type" => "text/csv"], body)
    end

    return json_response(404, Dict("error" => "not found"))
end

println("Starting Julia Analytics Service on port $PORT")
HTTP.serve(route, "0.0.0.0", PORT)
