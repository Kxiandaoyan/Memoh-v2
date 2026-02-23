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
	ID        string    `json:"id,omitempty"`
	BotID     string    `json:"bot_id,omitempty"`
	ToolName  string    `json:"tool_name"`
	Enabled   bool      `json:"enabled"`
	Priority  int       `json:"priority"`
	Category  string    `json:"category"`
	Tier      string    `json:"tier,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
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
		SELECT id, bot_id, tool_name, enabled, priority, category, tier, created_at, updated_at
		FROM builtin_tool_configs WHERE bot_id = $1
		ORDER BY priority ASC, tool_name ASC`, botID)
	if err != nil {
		return nil, fmt.Errorf("query builtin tool configs: %w", err)
	}
	defer rows.Close()
	var configs []BuiltinToolConfig
	for rows.Next() {
		var c BuiltinToolConfig
		if err := rows.Scan(&c.ID, &c.BotID, &c.ToolName, &c.Enabled, &c.Priority, &c.Category, &c.Tier, &c.CreatedAt, &c.UpdatedAt); err != nil {
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
		if c.ToolName == "" {
			return fmt.Errorf("tool_name required")
		}
		if c.Category == "" {
			c.Category = "other"
		}
		tier := c.Tier
		if tier == "" {
			tier = "core"
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO builtin_tool_configs (bot_id, tool_name, enabled, priority, category, tier)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (bot_id, tool_name) DO UPDATE SET enabled = EXCLUDED.enabled, priority = EXCLUDED.priority, tier = EXCLUDED.tier, updated_at = now()`,
			botID, c.ToolName, c.Enabled, c.Priority, c.Category, tier); err != nil {
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
		{"knowledge_read", "knowledge"},
		{"knowledge_write", "knowledge"},
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

// GetToolTiers returns a map of toolName→tier for a bot.
func (s *BuiltinToolConfigService) GetToolTiers(ctx context.Context, botID string) (map[string]string, error) {
	if botID == "" {
		return nil, fmt.Errorf("bot_id cannot be empty")
	}
	rows, err := s.pool.Query(ctx, `SELECT tool_name, tier FROM builtin_tool_configs WHERE bot_id = $1`, botID)
	if err != nil {
		return nil, fmt.Errorf("query tool tiers: %w", err)
	}
	defer rows.Close()
	tiers := make(map[string]string)
	for rows.Next() {
		var name, tier string
		if err := rows.Scan(&name, &tier); err != nil {
			return nil, fmt.Errorf("scan tool tier: %w", err)
		}
		tiers[name] = tier
	}
	return tiers, rows.Err()
}

// GetExtendedTools returns enabled tools with tier='extended' for a bot.
func (s *BuiltinToolConfigService) GetExtendedTools(ctx context.Context, botID string) ([]BuiltinToolConfig, error) {
	if botID == "" {
		return nil, fmt.Errorf("bot_id cannot be empty")
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, bot_id, tool_name, enabled, priority, category, tier, created_at, updated_at
		 FROM builtin_tool_configs WHERE bot_id = $1 AND tier = 'extended' AND enabled = true
		 ORDER BY tool_name`, botID)
	if err != nil {
		return nil, fmt.Errorf("query extended tools: %w", err)
	}
	defer rows.Close()
	var configs []BuiltinToolConfig
	for rows.Next() {
		var c BuiltinToolConfig
		if err := rows.Scan(&c.ID, &c.BotID, &c.ToolName, &c.Enabled, &c.Priority, &c.Category, &c.Tier, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan extended tool: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}
