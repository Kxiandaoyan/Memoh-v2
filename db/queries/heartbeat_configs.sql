-- name: CreateHeartbeatConfig :one
INSERT INTO heartbeat_configs (bot_id, enabled, interval_seconds, prompt, event_triggers)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetHeartbeatConfig :one
SELECT * FROM heartbeat_configs WHERE id = $1;

-- name: ListHeartbeatConfigsByBot :many
SELECT * FROM heartbeat_configs WHERE bot_id = $1 ORDER BY created_at;

-- name: ListEnabledHeartbeatConfigs :many
SELECT * FROM heartbeat_configs WHERE enabled = true ORDER BY created_at;

-- name: UpdateHeartbeatConfig :one
UPDATE heartbeat_configs
SET enabled = $2,
    interval_seconds = $3,
    prompt = $4,
    event_triggers = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteHeartbeatConfig :exec
DELETE FROM heartbeat_configs WHERE id = $1;
