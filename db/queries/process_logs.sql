-- name: CreateProcessLog :one
INSERT INTO process_logs (
    bot_id,
    chat_id,
    trace_id,
    user_id,
    channel,
    step,
    level,
    message,
    data,
    duration_ms
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetRecentProcessLogs :many
SELECT * FROM process_logs
WHERE bot_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetProcessLogsByTrace :many
SELECT * FROM process_logs
WHERE trace_id = $1
ORDER BY created_at ASC;

-- name: GetProcessLogsByChat :many
SELECT * FROM process_logs
WHERE bot_id = $1 AND chat_id = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: GetProcessLogsByStep :many
SELECT * FROM process_logs
WHERE bot_id = $1 AND step = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: GetProcessLogsByTimeRange :many
SELECT * FROM process_logs
WHERE bot_id = $1
AND created_at >= $2
AND created_at <= $3
ORDER BY created_at DESC
LIMIT $4;

-- name: GetProcessLogStats :many
SELECT
    step,
    COUNT(*) as count,
    AVG(duration_ms) as avg_duration_ms
FROM process_logs
WHERE bot_id = $1
AND created_at >= NOW() - INTERVAL '1 hour'
GROUP BY step
ORDER BY count DESC;

-- name: DeleteProcessLogsOlderThan :exec
DELETE FROM process_logs
WHERE created_at < $1;

-- name: GetProcessLogsByChatASC :many
SELECT * FROM process_logs
WHERE bot_id = $1 AND chat_id = $2
ORDER BY created_at ASC;
