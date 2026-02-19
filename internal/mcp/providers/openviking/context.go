package openviking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	dbsqlc "github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
)

const (
	contextQueryTimeout = 8 * time.Second
	cacheTTL            = 120 * time.Second
	maxContextResults   = 3
)

type cacheEntry struct {
	result    string
	expiresAt time.Time
}

// ContextLoader provides lightweight OpenViking context for conversation
// injection. It runs a quick semantic search and returns L0 abstracts.
type ContextLoader struct {
	execRunner ExecRunner
	queries    *dbsqlc.Queries
	logger     *slog.Logger

	mu    sync.Mutex
	cache map[string]cacheEntry
}

func NewContextLoader(log *slog.Logger, execRunner ExecRunner, queries *dbsqlc.Queries) *ContextLoader {
	if log == nil {
		log = slog.Default()
	}
	return &ContextLoader{
		execRunner: execRunner,
		queries:    queries,
		logger:     log.With(slog.String("component", "ov_context_loader")),
		cache:      make(map[string]cacheEntry),
	}
}

func (c *ContextLoader) isEnabled(ctx context.Context, botID string) bool {
	if c.queries == nil {
		c.logger.Debug("openviking.context.isEnabled: disabled", slog.String("reason", "queries is nil"))
		return false
	}
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		c.logger.Debug("openviking.context.isEnabled: disabled", slog.Any("reason", err))
		return false
	}
	row, err := c.queries.GetBotPrompts(ctx, botUUID)
	if err != nil {
		c.logger.Debug("openviking.context.isEnabled: disabled", slog.Any("reason", err))
		return false
	}
	if !row.EnableOpenviking {
		c.logger.Debug("openviking.context.isEnabled: disabled", slog.String("reason", "EnableOpenviking is false"))
		return false
	}
	return true
}

// LoadContext runs a quick OpenViking search for the given query and returns
// formatted L0 abstracts suitable for injection as a system message.
// Returns empty string if OpenViking is disabled, has no data, or times out.
func (c *ContextLoader) LoadContext(ctx context.Context, botID, query string) string {
	if !c.isEnabled(ctx, botID) {
		c.logger.Debug("openviking.LoadContext: disabled, skipping")
		return ""
	}
	if strings.TrimSpace(query) == "" {
		c.logger.Debug("openviking.LoadContext: empty query, skipping")
		return ""
	}

	cacheKey := botID + ":" + query
	c.mu.Lock()
	if entry, ok := c.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.result
	}
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, contextQueryTimeout)
	defer cancel()

	script := fmt.Sprintf(`import openviking as ov, json, os
if not os.path.isdir('%s'):
    print(json.dumps({"items": []}))
else:
    client = ov.SyncOpenViking(path='%s', config_file='%s')
    client.initialize()
    try:
        results = client.find(%s, limit=%d)
        items = []
        for r in results.resources:
            try:
                ab = client.abstract(r.uri)
            except Exception:
                ab = ""
            items.append({"uri": r.uri, "score": round(r.score, 4), "abstract": ab})
        print(json.dumps({"items": items}))
    finally:
        client.close()`,
		ovDataPath, ovDataPath, ovConfPath,
		pyStr(query), maxContextResults)

	result, err := c.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{shellCmd, shellFlag, fmt.Sprintf("python3 -c %s", shellQuote(script))},
		WorkDir: "/data",
	})
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no running task found") || strings.Contains(errMsg, "not found") {
			c.logger.Debug("openviking.LoadContext: container not running, skipping",
				slog.String("bot_id", botID))
			return ""
		}
		c.logger.Warn("openviking.LoadContext: exec failed", slog.String("bot_id", botID), slog.Any("error", err))
		return ""
	}
	if result.ExitCode != 0 {
		c.logger.Warn("openviking.LoadContext: exec failed",
			slog.String("bot_id", botID),
			slog.Any("exit_code", result.ExitCode),
			slog.String("stderr", truncate(result.Stderr, 200)))
		return ""
	}

	stdout := strings.TrimSpace(result.Stdout)
	if stdout == "" {
		c.logger.Debug("openviking.LoadContext: empty stdout")
		return ""
	}

	var parsed struct {
		Items []struct {
			URI      string  `json:"uri"`
			Score    float64 `json:"score"`
			Abstract string  `json:"abstract"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		c.logger.Warn("openviking.LoadContext: JSON parse failed", slog.Any("error", err))
		return ""
	}
	if len(parsed.Items) == 0 {
		c.logger.Debug("openviking.LoadContext: no results")
		return ""
	}

	var sb strings.Builder
	sb.WriteString("OpenViking context (relevant knowledge base entries):\n")
	for _, item := range parsed.Items {
		abstract := strings.TrimSpace(item.Abstract)
		if abstract == "" {
			abstract = "(no abstract available)"
		}
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", item.URI, abstract))
	}

	text := strings.TrimSpace(sb.String())

	c.mu.Lock()
	c.cache[cacheKey] = cacheEntry{result: text, expiresAt: time.Now().Add(cacheTTL)}
	if len(c.cache) > 200 {
		now := time.Now()
		for k, v := range c.cache {
			if now.After(v.expiresAt) {
				delete(c.cache, k)
			}
		}
	}
	c.mu.Unlock()

	return text
}
