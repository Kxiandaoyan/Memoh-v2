-- name: GetBotChannelConfig :one
SELECT id, bot_id, channel_type, credentials, external_identity, self_identity, routing, capabilities, status, verified_at, created_at, updated_at
FROM bot_channel_configs
WHERE bot_id = $1 AND channel_type = $2
LIMIT 1;

-- name: GetBotChannelConfigByExternalIdentity :one
SELECT id, bot_id, channel_type, credentials, external_identity, self_identity, routing, capabilities, status, verified_at, created_at, updated_at
FROM bot_channel_configs
WHERE channel_type = $1 AND external_identity = $2
LIMIT 1;

-- name: UpsertBotChannelConfig :one
INSERT INTO bot_channel_configs (
  bot_id, channel_type, credentials, external_identity, self_identity, routing, capabilities, status, verified_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (bot_id, channel_type)
DO UPDATE SET
  credentials = EXCLUDED.credentials,
  external_identity = EXCLUDED.external_identity,
  self_identity = EXCLUDED.self_identity,
  routing = EXCLUDED.routing,
  capabilities = EXCLUDED.capabilities,
  status = EXCLUDED.status,
  verified_at = EXCLUDED.verified_at,
  updated_at = now()
RETURNING id, bot_id, channel_type, credentials, external_identity, self_identity, routing, capabilities, status, verified_at, created_at, updated_at;

-- name: ListBotChannelConfigsByType :many
SELECT id, bot_id, channel_type, credentials, external_identity, self_identity, routing, capabilities, status, verified_at, created_at, updated_at
FROM bot_channel_configs
WHERE channel_type = $1
ORDER BY created_at DESC;

-- name: GetUserChannelBinding :one
SELECT id, user_id, channel_type, config, created_at, updated_at
FROM user_channel_bindings
WHERE user_id = $1 AND channel_type = $2
LIMIT 1;

-- name: UpsertUserChannelBinding :one
INSERT INTO user_channel_bindings (user_id, channel_type, config)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, channel_type)
DO UPDATE SET
  config = EXCLUDED.config,
  updated_at = now()
RETURNING id, user_id, channel_type, config, created_at, updated_at;

-- name: ListUserChannelBindingsByPlatform :many
SELECT id, user_id, channel_type, config, created_at, updated_at
FROM user_channel_bindings
WHERE channel_type = $1
ORDER BY created_at DESC;

