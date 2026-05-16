-- Event embeddings for similarity search
CREATE TABLE event_embeddings (
    event_id UUID PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
    embedding FLOAT8[] NOT NULL,         -- vector embedding (e.g. 128-dim)
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_event_embeddings ON event_embeddings USING GIN (embedding);

-- User preference profiles for recommendations
CREATE TABLE user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    category_weights JSONB NOT NULL DEFAULT '{}',  -- { "music": 0.8, "sports": 0.2 }
    embedding FLOAT8[] NOT NULL DEFAULT '{}',       -- user vector embedding
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_preferences ON user_preferences USING GIN (embedding);

-- No-show predictions for overbooking intelligence
CREATE TABLE noshow_predictions (
    ticket_id UUID PRIMARY KEY REFERENCES tickets(id) ON DELETE CASCADE,
    probability FLOAT8 NOT NULL CHECK (probability >= 0 AND probability <= 1),
    risk_level VARCHAR(10) NOT NULL CHECK (risk_level IN ('low','medium','high')),
    predicted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_noshow_predictions_risk ON noshow_predictions(risk_level);

-- Refresh noshow predictions function
CREATE OR REPLACE FUNCTION refresh_noshow_predictions()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM noshow_predictions WHERE ticket_id = NEW.id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_refresh_noshow
    AFTER INSERT OR UPDATE OF status ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION refresh_noshow_predictions();
