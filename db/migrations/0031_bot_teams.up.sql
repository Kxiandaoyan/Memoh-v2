-- 0031_bot_teams
-- Add bot team management: teams, directed call relationships, and call audit logs

CREATE TABLE IF NOT EXISTS bot_teams (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    manager_bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_bot_teams_owner   ON bot_teams(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_bot_teams_manager ON bot_teams(manager_bot_id);

CREATE TABLE IF NOT EXISTS bot_team_members (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id          UUID NOT NULL REFERENCES bot_teams(id) ON DELETE CASCADE,
    source_bot_id    UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    target_bot_id    UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    role_description TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT bot_team_members_unique UNIQUE (team_id, source_bot_id, target_bot_id),
    CONSTRAINT bot_team_members_no_self CHECK (source_bot_id != target_bot_id)
);

CREATE INDEX IF NOT EXISTS idx_bot_team_members_team   ON bot_team_members(team_id);
CREATE INDEX IF NOT EXISTS idx_bot_team_members_source ON bot_team_members(source_bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_team_members_target ON bot_team_members(target_bot_id);

CREATE TABLE IF NOT EXISTS bot_call_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    caller_bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    target_bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    message       TEXT NOT NULL,
    result        TEXT,
    status        TEXT NOT NULL DEFAULT 'pending',
    call_depth    INT  NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ,
    CONSTRAINT bot_call_logs_status_check CHECK (status IN ('pending', 'completed', 'failed', 'timeout'))
);

CREATE INDEX IF NOT EXISTS idx_bot_call_logs_caller  ON bot_call_logs(caller_bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_call_logs_target  ON bot_call_logs(target_bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_call_logs_created ON bot_call_logs(caller_bot_id, created_at DESC);
