# Fraud Detection Module
# Advanced ML-based anomaly detection for ticket transfer patterns

module FraudDetection

using DataFrames
using Statistics
using Random

export IsolationForest, fit!, predict, train_test_split

struct IsolationForest
    n_estimators::Int
    max_samples::Int
    contamination::Float64
    trees::Vector
end

function IsolationForest(; n_estimators=100, max_samples=256, contamination=0.1)
    return IsolationForest(n_estimators, max_samples, contamination, [])
end

function fit!(model::IsolationForest, data::Matrix{Float64})
    model.trees = []
    n, d = size(data)
    
    for _ in 1:model.n_estimators
        sample_idx = rand(1:n, min(model.max_samples, n))
        sample = data[sample_idx, :]
        
        tree = build_itree(sample, 0)
        push!(model.trees, tree)
    end
    
    return model
end

function build_itree(data::Matrix{Float64}, depth::Int)
    n, d = size(data)
    
    if n <= 1 || depth >= 10
        return (leaf=true, size=n, depth=depth)
    end
    
    split_attr = rand(1:d)
    min_val = minimum(data[:, split_attr])
    max_val = maximum(data[:, split_attr])
    
    if min_val == max_val
        return (leaf=true, size=n, depth=depth)
    end
    
    split_val = min_val + rand() * (max_val - min_val)
    
    left_idx = data[:, split_attr] .< split_val
    right_idx = .!left_idx
    
    left_child = build_itree(data[left_idx, :], depth + 1)
    right_child = build_itree(data[right_idx, :], depth + 1)
    
    return (leaf=false, split_attr=split_attr, split_val=split_val,
            left=left_child, right=right_child, depth=depth)
end

function path_length(tree, sample::Vector{Float64})
    if tree.leaf
        return tree.depth + c_factor(tree.size)
    end
    
    if sample[tree.split_attr] < tree.split_val
        return path_length(tree.left, sample)
    else
        return path_length(tree.right, sample)
    end
end

function c_factor(n::Int)
    if n <= 1
        return 0.0
    end
    return 2.0 * (log(n - 1) + 0.5772156649) - 2.0 * (n - 1) / n
end

function anomaly_score(model::IsolationForest, sample::Vector{Float64})
    avg_path = mean([path_length(tree, sample) for tree in model.trees])
    return 2.0^(-avg_path / c_factor(length(model.trees)))
end

function predict(model::IsolationForest, data::Matrix{Float64})
    scores = [anomaly_score(model, data[i, :]) for i in 1:size(data, 1)]
    threshold = quantile(scores, 1.0 - model.contamination)
    return scores .> threshold, scores
end

function train_test_split(data::Matrix{Float64}, ratio::Float64=0.8)
    n = size(data, 1)
    idx = shuffle(1:n)
    split = Int(floor(n * ratio))
    return data[idx[1:split], :], data[idx[split+1:end], :]
end

# Feature engineering for ticket transfer data
function extract_features(tickets_df::DataFrame, transfers_df::DataFrame)
    features = Float64[]
    labels = Int[]
    
    # Group by event
    if nrow(transfers_df) > 0 && nrow(tickets_df) > 0
        event_groups = groupby(transfers_df, :event_id)
        
        for (key, group) in pairs(event_groups)
            n_transfers = nrow(group)
            n_tickets = nrow(filter(:event_id => e -> e == key.event_id, tickets_df))
            
            if n_tickets > 0
                push!(features, Float64[n_transfers / n_tickets, n_transfers, n_tickets])
            end
        end
    end
    
    return length(features) > 0 ? hcat(features...) : zeros(3, 1)
end

end # module
