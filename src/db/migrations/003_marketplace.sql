-- Dynamic pricing tiers per event
CREATE TABLE pricing_tiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    tier_name VARCHAR(100) NOT NULL,
    price NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    capacity INT NOT NULL CHECK (capacity >= 0),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pricing_tiers_event ON pricing_tiers(event_id);

-- Resale marketplace listings
CREATE TABLE resale_listings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    seller_id UUID NOT NULL REFERENCES users(id),
    asking_price NUMERIC(10,2) NOT NULL CHECK (asking_price > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','sold','cancelled','expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '7 days')
);

CREATE INDEX idx_resale_listings_ticket ON resale_listings(ticket_id);
CREATE INDEX idx_resale_listings_status ON resale_listings(status);
CREATE INDEX idx_resale_listings_expires ON resale_listings(expires_at);

-- Resale transaction log with ZK transfer proof
CREATE TABLE resale_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id UUID NOT NULL REFERENCES resale_listings(id) ON DELETE CASCADE,
    buyer_id UUID NOT NULL REFERENCES users(id),
    price NUMERIC(10,2) NOT NULL CHECK (price > 0),
    transfer_proof TEXT,              -- ZK proof of valid transfer
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_resale_transactions_listing ON resale_transactions(listing_id);
CREATE INDEX idx_resale_transactions_buyer ON resale_transactions(buyer_id);
