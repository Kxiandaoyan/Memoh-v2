-- name: CreateTeam :one
INSERT INTO bot_teams (owner_user_id, name, manager_bot_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTeamByID :one
SELECT * FROM bot_teams WHERE id = $1;

-- name: GetTeamByManagerBot :one
SELECT * FROM bot_teams WHERE manager_bot_id = $1;

-- name: ListTeamsByOwner :many
SELECT * FROM bot_teams WHERE owner_user_id = $1 ORDER BY created_at DESC;

-- name: UpdateTeamManagerBot :exec
UPDATE bot_teams SET manager_bot_id = $2, updated_at = now() WHERE id = $1;

-- name: DeleteTeam :exec
DELETE FROM bot_teams WHERE id = $1;

-- name: AddTeamMember :one
INSERT INTO bot_team_members (team_id, source_bot_id, target_bot_id, role_description)
VALUES ($1, $2, $3, $4)
ON CONFLICT (team_id, source_bot_id, target_bot_id) DO UPDATE SET role_description = EXCLUDED.role_description
RETURNING *;

-- name: RemoveTeamMember :exec
DELETE FROM bot_team_members WHERE id = $1;

-- name: ListTeamMembersByTeam :many
SELECT btm.id, btm.team_id, btm.source_bot_id, btm.target_bot_id, btm.role_description, btm.created_at,
       b.display_name AS target_display_name, b.metadata AS target_metadata
FROM bot_team_members btm
JOIN bots b ON b.id = btm.target_bot_id
WHERE btm.team_id = $1
ORDER BY btm.created_at ASC;

-- name: ListCallableTargets :many
SELECT btm.id, btm.target_bot_id, btm.role_description,
       b.display_name AS target_display_name, b.metadata AS target_metadata
FROM bot_team_members btm
JOIN bots b ON b.id = btm.target_bot_id
WHERE btm.team_id = $1 AND btm.source_bot_id = $2;

-- name: ListAllCallableTargetsForBot :many
SELECT DISTINCT btm.target_bot_id, btm.role_description,
       b.display_name AS target_display_name, b.metadata AS target_metadata,
       bt.id AS team_id, bt.name AS team_name
FROM bot_team_members btm
JOIN bots b ON b.id = btm.target_bot_id
JOIN bot_teams bt ON bt.id = btm.team_id
WHERE btm.source_bot_id = $1;

-- name: ListAllTeamContextForBot :many
SELECT bt.id AS team_id, bt.name AS team_name,
       bt.manager_bot_id,
       mb.display_name AS manager_display_name,
       btm.source_bot_id, btm.target_bot_id, btm.role_description,
       b.display_name AS target_display_name,
       b.metadata AS target_metadata
FROM bot_teams bt
JOIN bot_team_members btm ON btm.team_id = bt.id
JOIN bots b ON b.id = btm.target_bot_id
LEFT JOIN bots mb ON mb.id = bt.manager_bot_id
WHERE btm.source_bot_id = $1
ORDER BY bt.created_at ASC, btm.created_at ASC;

-- name: CheckCallPermission :one
SELECT COUNT(*) AS cnt
FROM bot_team_members btm
WHERE btm.source_bot_id = $1 AND btm.target_bot_id = $2;

-- name: CreateBotCallLog :one
INSERT INTO bot_call_logs (caller_bot_id, target_bot_id, message, status, call_depth)
VALUES ($1, $2, $3, 'pending', $4)
RETURNING *;

-- name: UpdateBotCallLog :exec
UPDATE bot_call_logs
SET status = $2, result = $3, completed_at = now()
WHERE id = $1;

-- name: ListBotCallLogs :many
SELECT * FROM bot_call_logs
WHERE caller_bot_id = $1 OR target_bot_id = $1
ORDER BY created_at DESC
LIMIT $2;
