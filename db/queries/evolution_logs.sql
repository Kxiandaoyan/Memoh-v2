-- name: CreateEvolutionLog :one
INSERT INTO evolution_logs (bot_id, heartbeat_config_id, trigger_reason, status)
VALUES ($1, $2, $3, 'running')
RETURNING *;

-- name: CompleteEvolutionLog :one
UPDATE evolution_logs
SET status = $2,
    changes_summary = $3,
    files_modified = $4,
    agent_response = $5,
    completed_at = now()
WHERE id = $1
RETURNING *;

-- name: ListEvolutionLogsByBot :many
SELECT * FROM evolution_logs
WHERE bot_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetEvolutionLog :one
SELECT * FROM evolution_logs
WHERE id = $1;

-- name: CountEvolutionLogsByBot :one
SELECT count(*) FROM evolution_logs
WHERE bot_id = $1;
