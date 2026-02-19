-- 0025_subagent_runs
-- Persists sub-agent run state to PostgreSQL so runs survive restarts,
-- are visible in the Web UI, and can be queried for auditing.

CREATE TABLE IF NOT EXISTS subagent_runs (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id         TEXT        NOT NULL UNIQUE,
  bot_id         TEXT        NOT NULL,
  name           TEXT        NOT NULL,
  task           TEXT        NOT NULL,
  status         TEXT        NOT NULL DEFAULT 'running'
                             CHECK (status IN ('running', 'completed', 'failed', 'aborted')),
  spawn_depth    INT         NOT NULL DEFAULT 0,
  parent_run_id  TEXT,
  result_summary TEXT,
  error_message  TEXT,
  started_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at       TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_subagent_runs_bot_id   ON subagent_runs (bot_id);
CREATE INDEX IF NOT EXISTS idx_subagent_runs_status   ON subagent_runs (status);
CREATE INDEX IF NOT EXISTS idx_subagent_runs_run_id   ON subagent_runs (run_id);
CREATE INDEX IF NOT EXISTS idx_subagent_runs_created  ON subagent_runs (bot_id, created_at DESC);
