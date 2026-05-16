module DB

using LibPQ

export get_connection, close_connection

const DB_HOST = get(ENV, "DB_HOST", "localhost")
const DB_PORT = get(ENV, "DB_PORT", "5432")
const DB_USER = get(ENV, "DB_USER", "aev")
const DB_PASS = get(ENV, "DB_PASSWORD", "aev_secret")
const DB_NAME = get(ENV, "DB_NAME", "aev_events")

const CONN_STRING = "host=$(DB_HOST) port=$(DB_PORT) user=$(DB_USER) password=$(DB_PASS) dbname=$(DB_NAME)"

function get_connection()
    try
        return LibPQ.Connection(CONN_STRING)
    catch e
        @warn "Database connection failed: $e"
        return nothing
    end
end

function close_connection(conn)
    if conn !== nothing
        close(conn)
    end
end

end
