package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BuiltinToolConfig represents per-bot configuration for a builtin tool.
type BuiltinToolConfig struct {
	ID        string
	BotID     string
	ToolName  string
	Enabled   bool
	Priority  int
	Category  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BuiltinToolConfigService manages builtin tool configurations per bot.
type BuiltinToolConfigService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewBuiltinToolConfigService creates a new service instance.
func NewBuiltinToolConfigService(pool *pgxpool.Pool, logger *slog.Logger) *BuiltinToolConfigService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BuiltinToolConfigService{
		pool:   pool,
		logger: logger.With(slog.String("service", "builtin_tool_config")),
	}
}

// GetByBot retrieves all tool configurations for a specific bot.
func (s *BuiltinToolConfigService) GetByBot(ctx context.Context, botID string) ([]BuiltinToolConfig, error) {
	if botID == "" {
		return nil, fmt.Errorf("bot_id cannot be empty")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, bot_id, tool_name, enabled, priority, category, created_at, updated_at
		FROM builtin_tool_configs WHERE bot_id = $1
		ORDER BY priority ASC, tool_name ASC`, botID)
	if err != nil {
		return nil, fmt.Errorf("query builtin tool configs: %w", err)
	}
	defer rows.Close()
	var configs []BuiltinToolConfig
	for rows.Next() {
		var c BuiltinToolConfig
		if err := rows.Scan(&c.ID, &c.BotID, &c.ToolName, &c.Enabled, &c.Priority, &c.Category, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan builtin tool config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpsertBatch updates or inserts multiple tool configurations in a transaction.
func (s *BuiltinToolConfigService) UpsertBatch(ctx context.Context, botID string, configs []BuiltinToolConfig) error {
	if botID == "" || len(configs) == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, c := range configs {
		if c.ToolName == "" || c.Category == "" {
			return fmt.Errorf("tool_name and category required for %s", c.ToolName)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO builtin_tool_configs (bot_id, tool_name, enabled, priority, category)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (bot_id, tool_name) DO UPDATE SET enabled = EXCLUDED.enabled, priority = EXCLUDED.priority, updated_at = now()`,
			botID, c.ToolName, c.Enabled, c.Priority, c.Category); err != nil {
			return fmt.Errorf("upsert %s: %w", c.ToolName, err)
		}
	}
	return tx.Commit(ctx)
}

// InitializeDefaults creates default builtin tool configurations for a bot if none exist.
func (s *BuiltinToolConfigService) InitializeDefaults(ctx context.Context, botID string) error {
	if botID == "" {
		return fmt.Errorf("bot_id cannot be empty")
	}
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM builtin_tool_configs WHERE bot_id = $1`, botID).Scan(&count); err != nil {
		return fmt.Errorf("check existing configs: %w", err)
	}
	if count > 0 {
		return nil
	}
	defaults := []struct{ name, category string }{
		{"read", "file"}, {"write", "file"}, {"list", "file"}, {"edit", "file"}, {"exec", "file"},
		{"send", "message"}, {"react", "message"},
		{"search_memory", "memory"},
		{"web_search", "web"},
		{"list_schedule", "schedule"}, {"get_schedule", "schedule"}, {"create_schedule", "schedule"}, {"update_schedule", "schedule"}, {"delete_schedule", "schedule"},
		{"lookup_channel_user", "directory"},
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, d := range defaults {
		if _, err := tx.Exec(ctx, `
			INSERT INTO builtin_tool_configs (bot_id, tool_name, enabled, priority, category)
			VALUES ($1, $2, true, 100, $3) ON CONFLICT (bot_id, tool_name) DO NOTHING`,
			botID, d.name, d.category); err != nil {
			return fmt.Errorf("insert default %s: %w", d.name, err)
		}
	}
	return tx.Commit(ctx)
}

// DeleteByBot deletes all builtin tool configurations for a specific bot.
func (s *BuiltinToolConfigService) DeleteByBot(ctx context.Context, botID string) error {
	if botID == "" {
		return fmt.Errorf("bot_id cannot be empty")
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM builtin_tool_configs WHERE bot_id = $1`, botID)
	return err
}
