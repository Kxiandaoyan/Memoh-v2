-- name: GetConversationSummary :one
SELECT * FROM conversation_summaries
WHERE bot_id = $1 AND chat_id = $2;

-- name: UpsertConversationSummary :one
INSERT INTO conversation_summaries (bot_id, chat_id, summary, message_count)
VALUES ($1, $2, $3, $4)
ON CONFLICT (bot_id, chat_id) DO UPDATE
SET summary = EXCLUDED.summary,
    message_count = conversation_summaries.message_count + EXCLUDED.message_count,
    updated_at = now()
RETURNING *;

-- name: DeleteConversationSummary :exec
DELETE FROM conversation_summaries
WHERE bot_id = $1 AND chat_id = $2;
