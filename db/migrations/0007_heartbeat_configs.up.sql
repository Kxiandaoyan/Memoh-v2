-- 0007_heartbeat_configs
-- Per-bot heartbeat configuration for periodic and event-driven proactive triggers.

CREATE TABLE IF NOT EXISTS heartbeat_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    interval_seconds INTEGER NOT NULL DEFAULT 0,
    prompt TEXT NOT NULL DEFAULT '',
    event_triggers JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_heartbeat_configs_bot_id ON heartbeat_configs(bot_id);
CREATE INDEX IF NOT EXISTS idx_heartbeat_configs_enabled ON heartbeat_configs(enabled);
