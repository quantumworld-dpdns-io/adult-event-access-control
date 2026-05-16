using HTTP
using JSON
using DataFrames
using LibPQ
using Dates
using Statistics

const DB_HOST = get(ENV, "DB_HOST", "localhost")
const DB_PORT = get(ENV, "DB_PORT", "5432")
const DB_USER = get(ENV, "DB_USER", "aev")
const DB_PASS = get(ENV, "DB_PASSWORD", "aev_secret")
const DB_NAME = get(ENV, "DB_NAME", "aev_events")
const PORT = parse(Int, get(ENV, "PORT", "8090"))

const CONN_STRING = "host=$(DB_HOST) port=$(DB_PORT) user=$(DB_USER) password=$(DB_PASS) dbname=$(DB_NAME)"

function get_db()
    try
        return LibPQ.Connection(CONN_STRING)
    catch e
        @warn "DB connection failed, using sample data" exception=e
        return nothing
    end
end

function fetch_tickets(conn)
    if conn === nothing
        return DataFrame(
            event_id = String[],
            status = String[],
            checked_in_at = Union{Missing,DateTime}[],
            created_at = DateTime[]
        )
    end
    
    result = execute(conn, """
        SELECT t.id, t.event_id, t.status, t.checked_in_at, t.issued_at,
               e.title as event_title, e.start_time
        FROM tickets t
        JOIN events e ON e.id = t.event_id
        ORDER BY t.issued_at DESC
    """)
    return DataFrame(result)
end

function fetch_transfers(conn)
    if conn === nothing
        return DataFrame(
            ticket_id = String[],
            from_user_id = String[],
            to_user_id = String[],
            created_at = DateTime[]
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

# Anomaly detection: Isolation Forest (simplified)
function detect_anomalies(tickets_df, transfers_df)
    results = Dict[]
    
    # Metric 1: Transfer ratio per event
    if nrow(transfers_df) > 0 && nrow(tickets_df) > 0
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
    
    # Metric 2: Check-in velocity (rapid consecutive check-ins)
    if nrow(tickets_df) > 0
        checked_in = filter(:status => s -> s == "checked_in", tickets_df)
        if nrow(checked_in) > 1
            times = sort(checked_in[!, :checked_in_at])
            gaps = diff(times ./ 1000)  # seconds between check-ins
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
    
    return results
end

# Sybil resistance scoring
function sybil_score(conn, user_id::String)
    # Count of distinct events attended (clustering signal)
    # World ID uniqueness is the primary anti-Sybil mechanism
    if conn === nothing
        return Dict("sybil_score" => 0.0, "risk_level" => "low")
    end
    
    result = execute(conn, """
        SELECT COUNT(DISTINCT event_id) as event_count,
               COUNT(*) as ticket_count,
               BOOL_OR(wp.nullifier_hash IS NOT NULL) as has_world_id
        FROM tickets t
        LEFT JOIN world_id_proofs wp ON wp.user_id = t.owner_id
        WHERE t.owner_id = \$1
    """, [user_id])
    
    df = DataFrame(result)
    if nrow(df) == 0
        return Dict("sybil_score" => 1.0, "risk_level" => "high")
    end
    
    row = first(df)
    has_world_id = coalesce(row.has_world_id, false)
    event_count = coalesce(row.event_count, 0)
    
    # Score: 0 (trusted) to 1 (suspicious)
    score = has_world_id ? 0.0 : min(1.0, 0.3 + 0.1 * (1.0 / max(event_count, 1)))
    
    risk = score < 0.2 ? "low" : score < 0.5 ? "medium" : "high"
    
    return Dict(
        "sybil_score" => round(score, digits=3),
        "risk_level" => risk,
        "has_world_id" => has_world_id,
        "events_attended" => event_count
    )
end

# Routes
function route(req::HTTP.Request)
    target = string(req.target)
    method = req.method
    
    if method == "GET" && target == "/health"
        return HTTP.Response(200, JSON.json(Dict("status" => "ok", "service" => "aev-analytics")))
    end
    
    if method == "GET" && target == "/api/analytics/overview"
        conn = get_db()
        tickets = fetch_tickets(conn)
        transfers = fetch_transfers(conn)
        
        overview = Dict(
            "total_tickets" => nrow(tickets),
            "total_transfers" => nrow(transfers),
            "transfer_rate" => nrow(tickets) > 0 ? round(nrow(transfers) / nrow(tickets), digits=4) : 0.0,
            "checked_in" => nrow(filter(:status => s -> s == "checked_in", tickets)),
            "anomalies" => detect_anomalies(tickets, transfers)
        )
        
        if conn !== nothing
            close(conn)
        end
        return HTTP.Response(200, JSON.json(overview))
    end
    
    if method == "GET" && startswith(target, "/api/analytics/sybil/")
        user_id = split(target, "/")[end]
        conn = get_db()
        result = sybil_score(conn, user_id)
        if conn !== nothing
            close(conn)
        end
        return HTTP.Response(200, JSON.json(result))
    end
    
    if method == "GET" && target == "/api/analytics/anomalies"
        conn = get_db()
        tickets = fetch_tickets(conn)
        transfers = fetch_transfers(conn)
        if conn !== nothing
            close(conn)
        end
        return HTTP.Response(200, JSON.json(Dict("anomalies" => detect_anomalies(tickets, transfers))))
    end
    
    return HTTP.Response(404, JSON.json(Dict("error" => "not found")))
end

println("Starting Julia Analytics Service on port $PORT")
HTTP.serve(route, "0.0.0.0", PORT)
