CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table: multi-auth support
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    email VARCHAR(255) UNIQUE,
    display_name VARCHAR(255),
    avatar_url TEXT
);

-- Auth providers: OAuth / OIDC / SIWE / World ID
CREATE TABLE auth_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google','github','siwe','world_id'
    provider_id TEXT NOT NULL,     -- sub from OIDC, address from SIWE, nullifier_hash from World ID
    metadata JSONB DEFAULT '{}',   -- provider-specific data
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_id)
);

CREATE INDEX idx_auth_providers_user ON auth_providers(user_id);

-- Events
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organizer_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    venue_name VARCHAR(255),
    venue_address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    capacity INT NOT NULL CHECK (capacity > 0),
    min_age INT DEFAULT 18 CHECK (min_age >= 0),
    image_url TEXT,
    category VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','published','cancelled','completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_events_organizer ON events(organizer_id);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_start ON events(start_time);

-- Tickets (off-chain)
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id),
    owner_id UUID NOT NULL REFERENCES users(id),
    ticket_type VARCHAR(50) NOT NULL DEFAULT 'standard'
        CHECK (ticket_type IN ('standard','vip','early_bird')),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','checked_in','refunded','transferred','cancelled')),
    qr_secret UUID NOT NULL DEFAULT uuid_generate_v4(),
    zk_proof_commitment TEXT,        -- Noir proof commitment hash
    risc0_receipt_json TEXT,          -- RISC Zero receipt JSON (fallback)
    nullifier_hash VARCHAR(255),     -- World ID nullifier for anti-Sybil
    soulbound_token_address TEXT,    -- On-chain SBT contract address
    soulbound_token_id VARCHAR(255), -- On-chain SBT token ID
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    checked_in_at TIMESTAMPTZ,
    CONSTRAINT unique_event_ticket UNIQUE(event_id, owner_id)
);

CREATE INDEX idx_tickets_owner ON tickets(owner_id);
CREATE INDEX idx_tickets_event ON tickets(event_id);
CREATE INDEX idx_tickets_qr ON tickets(qr_secret);

-- Check-in log
CREATE TABLE check_in_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES tickets(id),
    event_id UUID NOT NULL REFERENCES events(id),
    checked_by UUID REFERENCES users(id),
    method VARCHAR(20) NOT NULL CHECK (method IN ('qr','nfc','manual')),
    zk_proof_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Transfer log for anti-transfer analysis
CREATE TABLE transfer_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES tickets(id),
    from_user_id UUID NOT NULL REFERENCES users(id),
    to_user_id UUID NOT NULL REFERENCES users(id),
    from_proof_commitment TEXT,
    to_proof_commitment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transfer_ticket ON transfer_log(ticket_id);

-- Analytics / fraud detection materialised view
CREATE MATERIALIZED VIEW ticket_transfer_stats AS
SELECT
    e.id AS event_id,
    e.title AS event_title,
    COUNT(t.id) AS total_tickets,
    COUNT(tl.id) AS total_transfers,
    COUNT(tl.id)::FLOAT / NULLIF(COUNT(t.id), 0) AS transfer_ratio,
    COUNT(*) FILTER (WHERE t.status = 'checked_in') AS checked_in_count
FROM events e
LEFT JOIN tickets t ON t.event_id = e.id
LEFT JOIN transfer_log tl ON tl.ticket_id = t.id
GROUP BY e.id, e.title;

CREATE UNIQUE INDEX idx_ticket_transfer_stats_event ON ticket_transfer_stats(event_id);

-- Refresh function for materialized view
CREATE OR REPLACE FUNCTION refresh_ticket_transfer_stats()
RETURNS TRIGGER AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY ticket_transfer_stats;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER refresh_stats_on_transfer
    AFTER INSERT OR UPDATE ON transfer_log
    FOR EACH STATEMENT EXECUTE FUNCTION refresh_ticket_transfer_stats();
