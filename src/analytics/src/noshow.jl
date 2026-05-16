module NoShow

using DataFrames
using Dates

export predict_noshow, event_noshow_risk, high_risk_tickets

function predict_noshow(ticket_id::String, conn)
    if conn === nothing
        return Dict("ticket_id" => ticket_id, "probability" => 0.1, "risk_level" => "low")
    end

    result = execute(conn, """
        SELECT t.id AS ticket_id, t.owner_id, t.event_id, t.issued_at, t.ticket_type,
               e.start_time, e.category,
               EXTRACT(DOW FROM e.start_time) AS day_of_week,
               EXTRACT(HOUR FROM e.start_time) AS hour_of_day
        FROM tickets t
        JOIN events e ON e.id = t.event_id
        WHERE t.id = \$1
    """, [ticket_id])
    df = DataFrame(result)
    if nrow(df) == 0
        return Dict("ticket_id" => ticket_id, "probability" => 0.0, "risk_level" => "unknown")
    end
    row = first(df)

    event_id = row.event_id
    owner_id = row.owner_id
    start_time = row.start_time
    issued_at = row.issued_at
    day_of_week = coalesce(get(row, :day_of_week, 0), 0)
    hour_of_day = coalesce(get(row, :hour_of_day, 12), 12)

    prob = 0.0

    hours_before = Dates.value(Dates.DateTime(start_time) - Dates.DateTime(issued_at)) / 3600000.0
    if hours_before < 24
        prob += 0.3
    elseif hours_before < 72
        prob += 0.15
    end

    if day_of_week in (1, 2, 3, 4)
        prob += 0.05
    end
    if hour_of_day < 8 || hour_of_day > 22
        prob += 0.05
    end

    hist_result = execute(conn, """
        SELECT COUNT(*) AS total_attended,
               SUM(CASE WHEN t.status = 'checked_in' THEN 1 ELSE 0 END) AS checked_in_count
        FROM tickets t
        WHERE t.owner_id = \$1 AND t.status IN ('active', 'checked_in')
    """, [owner_id])
    hist_df = DataFrame(hist_result)
    if nrow(hist_df) > 0
        total = coalesce(first(hist_df).total_attended, 0)
        checked = coalesce(first(hist_df).checked_in_count, 0)
        if total > 0
            attendance_rate = checked / total
            prob += (1.0 - attendance_rate) * 0.3
        end
    end

    event_hist = execute(conn, """
        SELECT COUNT(*) AS total,
               SUM(CASE WHEN status = 'checked_in' THEN 1 ELSE 0 END) AS checked_in
        FROM tickets WHERE event_id = \$1
    """, [event_id])
    event_df = DataFrame(event_hist)
    if nrow(event_df) > 0
        total = coalesce(first(event_df).total, 0)
        checked = coalesce(first(event_df).checked_in, 0)
        if total > 0
            event_noshow = 1.0 - (checked / total)
            prob += event_noshow * 0.2
        end
    end

    prob = min(1.0, max(0.0, prob))
    risk_level = prob < 0.3 ? "low" : prob < 0.6 ? "medium" : "high"

    return Dict(
        "ticket_id" => ticket_id,
        "probability" => round(prob, digits=3),
        "risk_level" => risk_level
    )
end

function event_noshow_risk(event_id::String, conn)
    if conn === nothing
        return Dict("event_id" => event_id, "average_risk" => 0.0, "total_tickets" => 0)
    end

    result = execute(conn, """
        SELECT id AS ticket_id FROM tickets
        WHERE event_id = \$1 AND status NOT IN ('cancelled', 'refunded')
    """, [event_id])
    df = DataFrame(result)

    if nrow(df) == 0
        return Dict("event_id" => event_id, "average_risk" => 0.0, "total_tickets" => 0)
    end

    risks = Float64[]
    high_risk_count = 0
    for row in eachrow(df)
        pred = predict_noshow(row.ticket_id, conn)
        p = pred["probability"]
        push!(risks, p)
        if p >= 0.6
            high_risk_count += 1
        end
    end

    avg_risk = length(risks) > 0 ? mean(risks) : 0.0
    return Dict(
        "event_id" => event_id,
        "average_risk" => round(avg_risk, digits=3),
        "total_tickets" => nrow(df),
        "high_risk_tickets" => high_risk_count,
        "risk_distribution" => Dict(
            "low" => count(r -> r < 0.3, risks),
            "medium" => count(r -> 0.3 <= r < 0.6, risks),
            "high" => high_risk_count
        )
    )
end

function high_risk_tickets(event_id::String, threshold::Float64, conn)
    if conn === nothing
        return DataFrame(ticket_id = String[], probability = Float64[], risk_level = String[])
    end

    result = execute(conn, """
        SELECT id AS ticket_id FROM tickets
        WHERE event_id = \$1 AND status NOT IN ('cancelled', 'refunded')
    """, [event_id])
    df = DataFrame(result)

    high_risk = []
    for row in eachrow(df)
        pred = predict_noshow(row.ticket_id, conn)
        if pred["probability"] >= threshold
            push!(high_risk, (row.ticket_id, pred["probability"], pred["risk_level"]))
        end
    end

    if length(high_risk) == 0
        return DataFrame(ticket_id = String[], probability = Float64[], risk_level = String[])
    end

    return DataFrame(
        ticket_id = [h[1] for h in high_risk],
        probability = [h[2] for h in high_risk],
        risk_level = [h[3] for h in high_risk]
    )
end

end
