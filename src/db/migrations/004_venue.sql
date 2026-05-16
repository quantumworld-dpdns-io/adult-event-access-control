-- Venue sections for heatmap
CREATE TABLE venue_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    section_name TEXT NOT NULL,
    capacity INT NOT NULL CHECK (capacity > 0),
    sort_order INT DEFAULT 0
);

-- Alerts table
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    message TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info' CHECK (severity IN ('info','warning','critical')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- LISTEN/NOTIFY triggers
CREATE OR REPLACE FUNCTION notify_venue_update()
RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('venue_update', json_build_object(
        'event_id', NEW.event_id,
        'type', TG_ARGV[0],
        'timestamp', now()
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER alert_notify AFTER INSERT ON alerts
    FOR EACH ROW EXECUTE FUNCTION notify_venue_update('alert');

CREATE OR REPLACE FUNCTION notify_alert_channel()
RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('alert_channel', row_to_json(NEW)::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER alert_channel_trigger AFTER INSERT ON alerts
    FOR EACH ROW EXECUTE FUNCTION notify_alert_channel();

-- Indexes
CREATE INDEX idx_venue_sections_event ON venue_sections(event_id);
CREATE INDEX idx_alerts_event ON alerts(event_id, created_at DESC);
