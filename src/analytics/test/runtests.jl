using Test
using DataFrames

include("../src/fraud_detection.jl")
using .FraudDetection

@testset "FraudDetection" begin
    @testset "IsolationForest construction" begin
        model = FraudDetection.IsolationForest(n_estimators=10, contamination=0.1)
        @test model.n_estimators == 10
        @test model.contamination == 0.1
        @test model.train_size == 0
        @test isempty(model.trees)
    end

    @testset "fit! and predict" begin
        data = [
            1.0 2.0
            2.0 3.0
            3.0 4.0
            10.0 10.0
            11.0 11.0
        ]
        model = FraudDetection.IsolationForest(n_estimators=20, contamination=0.2)
        model = FraudDetection.fit!(model, data)
        @test length(model.trees) == 20
        @test model.train_size == 5
        is_anomaly, scores = FraudDetection.predict(model, data)
        @test length(is_anomaly) == 5
        @test length(scores) == 5
    end

    @testset "train_test_split" begin
        data = rand(10, 3)
        train, test = FraudDetection.train_test_split(data, 0.7)
        @test size(train, 1) == 7
        @test size(test, 1) == 3
        @test size(train, 2) == 3
    end

    @testset "extract_features" begin
        tickets = DataFrame(
            event_id = ["e1", "e1", "e2"],
            status = ["active", "checked_in", "active"],
            checked_in_at = [missing, DateTime(2024, 1, 1, 12, 0, 0), missing],
            issued_at = [DateTime(2024, 1, 1), DateTime(2024, 1, 1), DateTime(2024, 1, 2)]
        )
        transfers = DataFrame(
            ticket_id = ["t1", "t3"],
            from_user_id = ["u1", "u3"],
            to_user_id = ["u2", "u4"],
            created_at = [DateTime(2024, 1, 1, 13, 0, 0), DateTime(2024, 1, 2, 14, 0, 0)],
            event_id = ["e1", "e2"]
        )
        features = FraudDetection.extract_features(tickets, transfers)
        @test size(features, 1) == 3
        @test size(features, 2) == 2
    end
end

@testset "Server helpers" begin
    include("../src/db.jl")
    using .DB

    @test DB.DB_HOST == get(ENV, "DB_HOST", "localhost")
    @test DB.DB_PORT == get(ENV, "DB_PORT", "5432")
    @test DB.DB_USER == get(ENV, "DB_USER", "aev")
    @test DB.DB_NAME == get(ENV, "DB_NAME", "aev_events")

    conn = DB.get_connection()
    if conn === nothing
        @info "No database available, DB tests skipped (expected in CI)"
    else
        DB.close_connection(conn)
        @test true
    end
end

println("All tests passed!")
