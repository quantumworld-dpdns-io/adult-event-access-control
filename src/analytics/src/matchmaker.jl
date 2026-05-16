module Matchmaker

using DataFrames
using LinearAlgebra

export recommend_events, event_similarity, user_embedding

function user_embedding(user_id::String, conn)
    if conn === nothing
        return Dict("user_id" => user_id, "categories" => Dict(), "event_ids" => [])
    end

    result = execute(conn, """
        SELECT e.category, e.id AS event_id, COUNT(*) AS weight
        FROM tickets t
        JOIN events e ON e.id = t.event_id
        WHERE t.owner_id = \$1 AND t.status IN ('active', 'checked_in')
        GROUP BY e.category, e.id
    """, [user_id])
    df = DataFrame(result)

    if nrow(df) == 0
        return Dict("user_id" => user_id, "categories" => Dict(), "event_ids" => [])
    end

    cat_weights = Dict{String, Float64}()
    event_ids = String[]
    for row in eachrow(df)
        cat = coalesce(get(row, :category, "uncategorized"), "uncategorized")
        weight = coalesce(get(row, :weight, 1), 1)
        cat_weights[cat] = get(cat_weights, cat, 0.0) + Float64(weight)
        push!(event_ids, row.event_id)
    end

    norm = sqrt(sum(v^2 for v in values(cat_weights)))
    if norm > 0
        for k in keys(cat_weights)
            cat_weights[k] /= norm
        end
    end

    return Dict(
        "user_id" => user_id,
        "categories" => cat_weights,
        "event_ids" => event_ids
    )
end

function recommend_events(user_id::String, conn; limit::Int=5)
    if conn === nothing
        return [Dict("event_id" => "", "score" => 0.0, "reason" => "no data")]
    end

    embed = user_embedding(user_id, conn)
    user_cats = embed["categories"]
    if length(user_cats) == 0
        result = execute(conn, """
            SELECT id, title, category, start_time
            FROM events
            WHERE status = 'published'
            ORDER BY start_time ASC
            LIMIT \$1
        """, [limit])
        df = DataFrame(result)
        events = [Dict(
            "event_id" => row.id,
            "title" => row.title,
            "category" => coalesce(get(row, :category, ""), ""),
            "score" => 0.0,
            "reason" => "popular"
        ) for row in eachrow(df)]
        return events
    end

    result = execute(conn, """
        SELECT id, title, category, start_time
        FROM events
        WHERE status = 'published' AND id NOT IN (
            SELECT event_id FROM tickets WHERE owner_id = \$1
        )
    """, [user_id])
    df = DataFrame(result)

    scored = []
    for row in eachrow(df)
        cat = coalesce(get(row, :category, "uncategorized"), "uncategorized")
        cat_score = get(user_cats, cat, 0.0)
        if cat_score > 0
            push!(scored, Dict(
                "event_id" => row.id,
                "title" => row.title,
                "category" => cat,
                "score" => round(cat_score, digits=3),
                "reason" => "matches your interest in $cat"
            ))
        end
    end

    sort!(scored, by = x -> x["score"], rev = true)
    return length(scored) > 0 ? scored[1:min(limit, length(scored))] : scored
end

function event_similarity(event_id::String, conn; limit::Int=5)
    if conn === nothing
        return [Dict("event_id" => "", "similarity" => 0.0)]
    end

    source_result = execute(conn, """
        SELECT category, start_time
        FROM events WHERE id = \$1
    """, [event_id])
    source_df = DataFrame(source_result)
    if nrow(source_df) == 0
        return Dict[]
    end
    source = first(source_df)
    source_cat = coalesce(get(source, :category, ""), "")
    source_time = source.start_time

    candidates = execute(conn, """
        SELECT e.id, e.title, e.category, e.start_time,
               COUNT(t.id) AS attendee_count
        FROM events e
        LEFT JOIN tickets t ON t.event_id = e.id AND t.status IN ('active', 'checked_in')
        WHERE e.id != \$1 AND e.status = 'published'
        GROUP BY e.id, e.title, e.category, e.start_time
    """, [event_id])
    cand_df = DataFrame(candidates)

    results = []
    for row in eachrow(cand_df)
        sim = 0.0
        cat = coalesce(get(row, :category, ""), "")

        if source_cat != "" && cat == source_cat
            sim += 0.5
        end

        time_diff = abs(Dates.value(Dates.DateTime(source_time) - Dates.DateTime(row.start_time)))
        days_diff = time_diff / 86400000.0
        if days_diff < 7
            sim += 0.3 * (1.0 - days_diff / 7.0)
        end

        push!(results, Dict(
            "event_id" => row.id,
            "title" => row.title,
            "category" => cat,
            "similarity" => round(sim, digits=3)
        ))
    end

    sort!(results, by = x -> x["similarity"], rev = true)
    return length(results) > 0 ? results[1:min(limit, length(results))] : results
end

end
