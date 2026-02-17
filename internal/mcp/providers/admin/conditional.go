package admin

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
)

// ConditionalExecutor wraps the admin Executor and only exposes tools
// when the requesting bot has is_privileged = true.
type ConditionalExecutor struct {
	inner   *Executor
	queries *sqlc.Queries
	logger  *slog.Logger
}

// NewConditionalExecutor wraps an admin executor with a privilege check.
func NewConditionalExecutor(log *slog.Logger, inner *Executor, queries *sqlc.Queries) *ConditionalExecutor {
	if log == nil {
		log = slog.Default()
	}
	return &ConditionalExecutor{
		inner:   inner,
		queries: queries,
		logger:  log.With(slog.String("provider", "admin_conditional")),
	}
}

func (c *ConditionalExecutor) isPrivileged(ctx context.Context, botID string) bool {
	if c.queries == nil || strings.TrimSpace(botID) == "" {
		return false
	}
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return false
	}
	privileged, err := c.queries.GetBotIsPrivileged(ctx, pgBotID)
	if err != nil {
		c.logger.Debug("failed to check is_privileged", slog.String("bot_id", botID), slog.Any("error", err))
		return false
	}
	return privileged
}

func (c *ConditionalExecutor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	if !c.isPrivileged(ctx, session.BotID) {
		return []mcpgw.ToolDescriptor{}, nil
	}
	return c.inner.ListTools(ctx, session)
}

func (c *ConditionalExecutor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if !c.isPrivileged(ctx, session.BotID) {
		return mcpgw.BuildToolErrorResult("this bot is not privileged"), nil
	}
	return c.inner.CallTool(ctx, session, toolName, arguments)
}
