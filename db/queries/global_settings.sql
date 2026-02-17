-- name: GetGlobalSetting :one
SELECT key, value, updated_at FROM global_settings WHERE key = $1;

-- name: UpsertGlobalSetting :one
INSERT INTO global_settings (key, value, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
RETURNING key, value, updated_at;

-- name: ListGlobalSettings :many
SELECT key, value, updated_at FROM global_settings ORDER BY key;
