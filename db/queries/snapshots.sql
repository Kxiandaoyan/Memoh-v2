-- name: InsertSnapshot :exec
INSERT INTO snapshots (id, container_id, parent_snapshot_id, snapshotter, digest)
VALUES (
  sqlc.arg(id),
  sqlc.arg(container_id),
  sqlc.arg(parent_snapshot_id),
  sqlc.arg(snapshotter),
  sqlc.arg(digest)
)
ON CONFLICT (id) DO NOTHING;

-- name: ListSnapshotsByContainerID :many
SELECT id, container_id, parent_snapshot_id, snapshotter, digest, created_at
FROM snapshots
WHERE container_id = sqlc.arg(container_id)
ORDER BY created_at DESC;

-- name: DeleteSnapshot :exec
DELETE FROM snapshots WHERE id = sqlc.arg(id);
