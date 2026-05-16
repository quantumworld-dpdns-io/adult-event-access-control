-- World ID proof log for audit
CREATE TABLE world_id_proofs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    nullifier_hash VARCHAR(255) NOT NULL,
    merkle_root VARCHAR(255),
    proof_json TEXT,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id)
);

CREATE INDEX idx_world_id_proofs_nullifier ON world_id_proofs(nullifier_hash);

-- Event capacity notification channel
CREATE OR REPLACE FUNCTION notify_capacity()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        'capacity_channel',
        json_build_object(
            'event_id', NEW.event_id,
            'current_occupancy', (SELECT COUNT(*) FROM tickets WHERE event_id = NEW.event_id AND status = 'checked_in')
        )::text
    );
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER checkin_capacity_notify
    AFTER UPDATE ON tickets
    FOR EACH ROW
    WHEN (NEW.status = 'checked_in' AND OLD.status != 'checked_in')
    EXECUTE FUNCTION notify_capacity();
