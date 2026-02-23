-- name: CreateSubagent :one
WITH revived AS (
  UPDATE subagents
  SET deleted = false, deleted_at = NULL,
      description = $2, messages = $4, metadata = $5, skills = $6, updated_at = now()
  WHERE bot_id = $3 AND name = $1 AND deleted = true
  RETURNING id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills
), inserted AS (
  INSERT INTO subagents (name, description, bot_id, messages, metadata, skills)
  SELECT $1, $2, $3, $4, $5, $6
  WHERE NOT EXISTS (SELECT 1 FROM revived)
    AND NOT EXISTS (SELECT 1 FROM subagents WHERE bot_id = $3 AND name = $1 AND deleted = false)
  RETURNING id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills
), updated AS (
  UPDATE subagents
  SET description = $2, messages = $4, metadata = $5, skills = $6, updated_at = now()
  WHERE bot_id = $3 AND name = $1 AND deleted = false
    AND NOT EXISTS (SELECT 1 FROM revived)
    AND NOT EXISTS (SELECT 1 FROM inserted)
  RETURNING id, name, description, created_at, updated_at, deleted, deleted_at, bot_id, messages, metadata, skills
)
SELECT * FROM revived
UNION ALL SELECT * FROM inserted
UNION ALL SELECT * FROM updated;

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

