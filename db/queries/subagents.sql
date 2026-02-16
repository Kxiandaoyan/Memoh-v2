-- name: CreateSubagent :one
INSERT INTO subagents (name, description, bot_id, messages, metadata, skills)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills;

-- name: GetSubagentByID :one
SELECT id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills
FROM subagents
WHERE id = $1 AND deleted = false;

-- name: ListSubagentsByBot :many
SELECT id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills
FROM subagents
WHERE bot_id = $1 AND deleted = false
ORDER BY created_at DESC;

-- name: UpdateSubagent :one
UPDATE subagents
SET name = $2,
    description = $3,
    metadata = $4,
    updated_at = now()
WHERE id = $1 AND deleted = false
RETURNING id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills;

-- name: UpdateSubagentMessages :one
UPDATE subagents
SET messages = $2,
    updated_at = now()
WHERE id = $1 AND deleted = false
RETURNING id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills;

-- name: UpdateSubagentSkills :one
UPDATE subagents
SET skills = $2,
    updated_at = now()
WHERE id = $1 AND deleted = false
RETURNING id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills;

-- name: SoftDeleteSubagent :exec
UPDATE subagents
SET deleted = true,
    deleted_at = now(),
    updated_at = now()
WHERE id = $1 AND deleted = false;

