module Pricing

using DataFrames
using Dates

export calculate_surge, demand_forecast

function calculate_surge(event_id::String, conn)
    if conn === nothing
        return Dict(
            "event_id" => event_id,
            "surge_multiplier" => 1.0,
            "base_price" => 0.0,
            "demand_level" => "unknown"
        )
    end

    result = execute(conn, """
        SELECT e.capacity, e.start_time,
               COALESCE(AVG(pt.price), 0.0) AS base_price,
               COUNT(t.id) AS tickets_sold
        FROM events e
        LEFT JOIN pricing_tiers pt ON pt.event_id = e.id
        LEFT JOIN tickets t ON t.event_id = e.id AND t.status != 'cancelled'
        WHERE e.id = \$1
        GROUP BY e.capacity, e.start_time
    """, [event_id])
    df = DataFrame(result)
    if nrow(df) == 0
        return Dict("event_id" => event_id, "surge_multiplier" => 1.0, "demand_level" => "unknown")
    end

    row = first(df)
    capacity = row.capacity
    base_price = coalesce(row.base_price, 0.0)
    tickets_sold = coalesce(row.tickets_sold, 0)
    start_time = row.start_time

    ratio = capacity > 0 ? tickets_sold / capacity : 0.0
    hours_until = Dates.value(Dates.DateTime(start_time) - now()) / 3600000.0

    surge = 1.0
    demand_level = "low"

    if ratio > 0.9
        surge = 2.0
        demand_level = "very_high"
    elseif ratio > 0.75
        surge = 1.5
        demand_level = "high"
    elseif ratio > 0.5
        surge = 1.2
        demand_level = "medium"
    end

    if hours_until < 24.0 && ratio > 0.5
        surge += 0.3
    elseif hours_until < 72.0 && ratio > 0.7
        surge += 0.2
    end

    return Dict(
        "event_id" => event_id,
        "surge_multiplier" => round(surge, digits=2),
        "base_price" => round(base_price, digits=2),
        "current_price" => round(base_price * surge, digits=2),
        "demand_level" => demand_level,
        "tickets_sold" => tickets_sold,
        "capacity" => capacity,
        "tickets_remaining" => capacity - tickets_sold,
        "hours_until_event" => round(hours_until, digits=1)
    )
end

function demand_forecast(event_id::String, conn)
    if conn === nothing
        return Dict("event_id" => event_id, "forecast" => [])
    end

    historical = execute(conn, """
        SELECT DATE(issued_at) AS sale_date, COUNT(*) AS sales
        FROM tickets
        WHERE event_id = \$1 AND status != 'cancelled'
        GROUP BY DATE(issued_at)
        ORDER BY sale_date
    """, [event_id])
    hist_df = DataFrame(historical)

    if nrow(hist_df) == 0
        return Dict("event_id" => event_id, "forecast" => [], "message" => "insufficient data")
    end

    recent_sales = hist_df[!, :sales]
    daily_avg = mean(recent_sales)
    daily_std = std(recent_sales)

    result = execute(conn, """
        SELECT capacity, COUNT(t.id) AS total_sold
        FROM events e
        LEFT JOIN tickets t ON t.event_id = e.id AND t.status != 'cancelled'
        WHERE e.id = \$1
        GROUP BY e.capacity
    """, [event_id])
    cap_df = DataFrame(result)
    capacity = nrow(cap_df) > 0 ? first(cap_df).capacity : 0
    total_sold = nrow(cap_df) > 0 ? coalesce(first(cap_df).total_sold, 0) : 0

    remaining_capacity = capacity - total_sold
    days_until_event = 7
    forecasted_sales = min(remaining_capacity, max(0, Int(round(daily_avg * days_until_event))))

    return Dict(
        "event_id" => event_id,
        "forecast" => [
            Dict(
                "period" => "next_7_days",
                "predicted_sales" => forecasted_sales,
                "remaining_capacity" => remaining_capacity,
                "confidence" => daily_std > 0 ? round(1.0 - daily_std / max(daily_avg, 1), digits=2) : 0.5
            )
        ],
        "daily_average_sales" => round(daily_avg, digits=1)
    )
end

end
