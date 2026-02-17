-- 0016_evolution_logs
-- Add evolution_logs table to track self-evolution history per bot

CREATE TABLE IF NOT EXISTS evolution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    heartbeat_config_id UUID REFERENCES heartbeat_configs(id) ON DELETE SET NULL,
    trigger_reason TEXT NOT NULL DEFAULT 'periodic',
    status TEXT NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed', 'skipped')),
    changes_summary TEXT,
    files_modified TEXT[],
    agent_response TEXT,
    started_at TIMESTAMPTZ DEFAULT now(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_evolution_logs_bot_id ON evolution_logs(bot_id);
CREATE INDEX IF NOT EXISTS idx_evolution_logs_bot_id_created_at ON evolution_logs(bot_id, created_at DESC);
