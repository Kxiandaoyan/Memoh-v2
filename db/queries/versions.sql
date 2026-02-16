-- name: ListVersionsByContainerID :many
SELECT * FROM container_versions WHERE container_id = sqlc.arg(container_id) ORDER BY version ASC;

-- name: NextVersion :one
SELECT COALESCE(MAX(version), 0) + 1 FROM container_versions WHERE container_id = sqlc.arg(container_id);

-- name: InsertVersion :one
INSERT INTO container_versions (id, container_id, snapshot_id, version)
VALUES (
  sqlc.arg(id),
  sqlc.arg(container_id),
  sqlc.arg(snapshot_id),
  sqlc.arg(version)
)
RETURNING *;

-- name: GetVersionSnapshotID :one
SELECT snapshot_id FROM container_versions WHERE container_id = sqlc.arg(container_id) AND version = sqlc.arg(version);
