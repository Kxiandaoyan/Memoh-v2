package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
)

const toolSearchPassive = "search_passive_messages"

type Executor struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewExecutor(log *slog.Logger, pool *pgxpool.Pool) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		pool:   pool,
		logger: log.With(slog.String("provider", "inbox_tool")),
	}
}

func (p *Executor) ListTools(_ context.Context, _ mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolSearchPassive,
			Description: "Search passive (non-mentioned) group messages the bot received but did not respond to.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":      map[string]any{"type": "string", "description": "Search keyword"},
					"start_time": map[string]any{"type": "string", "description": "ISO 8601 start time"},
					"end_time":   map[string]any{"type": "string", "description": "ISO 8601 end time"},
					"limit":      map[string]any{"type": "integer", "description": "Max results (default 20, max 100)"},
				},
				"required": []string{},
			},
		},
	}, nil
}

type passiveMessage struct {
	ID        string          `json:"id"`
	Platform  string          `json:"platform"`
	Content   json.RawMessage `json:"content"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

func (p *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if toolName != toolSearchPassive {
		return nil, mcpgw.ErrToolNotFound
	}
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}

	query := strings.TrimSpace(mcpgw.StringArg(arguments, "query"))
	limit := 20
	if v, ok, _ := mcpgw.IntArg(arguments, "limit"); ok && v > 0 {
		limit = v
		if limit > 100 {
			limit = 100
		}
	}

	sql := `SELECT id, channel_type, content, metadata, created_at
FROM bot_history_messages
WHERE bot_id = $1
  AND metadata->>'trigger_mode' = 'passive_sync'`
	args := []any{botID}
	argIdx := 2

	if query != "" {
		sql += fmt.Sprintf(` AND content::text ILIKE '%%' || $%d || '%%'`, argIdx)
		args = append(args, query)
		argIdx++
	}
	if v := strings.TrimSpace(mcpgw.StringArg(arguments, "start_time")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			sql += fmt.Sprintf(` AND created_at >= $%d`, argIdx)
			args = append(args, t)
			argIdx++
		}
	}
	if v := strings.TrimSpace(mcpgw.StringArg(arguments, "end_time")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			sql += fmt.Sprintf(` AND created_at <= $%d`, argIdx)
			args = append(args, t)
			argIdx++
		}
	}
	sql += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d`, argIdx)
	args = append(args, limit)

	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		p.logger.Warn("search passive failed", slog.Any("error", err))
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	defer rows.Close()

	var messages []map[string]any
	for rows.Next() {
		var m passiveMessage
		if err := rows.Scan(&m.ID, &m.Platform, &m.Content, &m.Metadata, &m.CreatedAt); err != nil {
			continue
		}
		entry := map[string]any{
			"id":         m.ID,
			"platform":   m.Platform,
			"created_at": m.CreatedAt.Format(time.RFC3339),
		}
		var content map[string]any
		if json.Unmarshal(m.Content, &content) == nil {
			entry["content"] = content
		}
		var meta map[string]any
		if json.Unmarshal(m.Metadata, &meta) == nil {
			entry["metadata"] = meta
		}
		messages = append(messages, entry)
	}
	if messages == nil {
		messages = []map[string]any{}
	}

	return mcpgw.BuildToolSuccessResult(map[string]any{
		"ok":       true,
		"bot_id":   botID,
		"count":    len(messages),
		"messages": messages,
	}), nil
}
