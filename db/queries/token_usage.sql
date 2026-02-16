-- name: RecordTokenUsage :one
INSERT INTO token_usage (bot_id, prompt_tokens, completion_tokens, total_tokens, model, source)
VALUES (@bot_id, @prompt_tokens, @completion_tokens, @total_tokens, @model, @source)
RETURNING *;

-- name: GetBotTokenTotal :one
SELECT
  COALESCE(SUM(prompt_tokens), 0)::bigint AS prompt_tokens,
  COALESCE(SUM(completion_tokens), 0)::bigint AS completion_tokens,
  COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens
FROM token_usage
WHERE bot_id = @bot_id;

-- name: GetBotTokenDailySeries :many
SELECT
  date_trunc('day', created_at)::date AS day,
  COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens,
  COALESCE(SUM(prompt_tokens), 0)::bigint AS prompt_tokens,
  COALESCE(SUM(completion_tokens), 0)::bigint AS completion_tokens
FROM token_usage
WHERE bot_id = @bot_id
  AND created_at >= @since
  AND created_at < @until
GROUP BY date_trunc('day', created_at)
ORDER BY day;

-- name: GetAllBotsTokenDailySeries :many
SELECT
  bot_id,
  date_trunc('day', created_at)::date AS day,
  COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens
FROM token_usage
WHERE created_at >= @since
  AND created_at < @until
GROUP BY bot_id, date_trunc('day', created_at)
ORDER BY day;

-- name: GetAllBotsTokenTotals :many
SELECT
  bot_id,
  COALESCE(SUM(prompt_tokens), 0)::bigint AS prompt_tokens,
  COALESCE(SUM(completion_tokens), 0)::bigint AS completion_tokens,
  COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens
FROM token_usage
GROUP BY bot_id;
