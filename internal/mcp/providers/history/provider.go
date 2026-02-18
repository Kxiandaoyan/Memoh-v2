package history

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	messagepkg "github.com/Kxiandaoyan/Memoh-v2/internal/message"
)

const (
	toolQueryHistory  = "query_history"
	defaultLimit      = 50
	maxLimit          = 200
	maxHoursBack      = 168 // 7 days
)

type Executor struct {
	messageService messagepkg.Service
	logger         *slog.Logger
}

func NewExecutor(log *slog.Logger, messageService messagepkg.Service) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		messageService: messageService,
		logger:         log.With(slog.String("provider", "history_tool")),
	}
}

func (p *Executor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	if p.messageService == nil {
		return []mcpgw.ToolDescriptor{}, nil
	}
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolQueryHistory,
			Description: "Query your conversation history. Returns recent messages across all channels (DM and group). Use this to review past conversations for evolution, reflection, or context.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"hours_back": map[string]any{
						"type":        "number",
						"description": fmt.Sprintf("How many hours back to search (1-%d). Default: 24", maxHoursBack),
					},
					"limit": map[string]any{
						"type":        "number",
						"description": fmt.Sprintf("Max messages to return (1-%d). Default: %d", maxLimit, defaultLimit),
					},
					"role": map[string]any{
						"type":        "string",
						"description": "Filter by role: 'user', 'assistant', or empty for all",
						"enum":        []string{"", "user", "assistant"},
					},
					"keyword": map[string]any{
						"type":        "string",
						"description": "Filter messages containing this keyword (case-insensitive)",
					},
				},
				"required": []string{},
			},
		},
	}, nil
}

func (p *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if toolName != toolQueryHistory {
		return nil, mcpgw.ErrToolNotFound
	}
	return p.callQueryHistory(ctx, session, arguments)
}

func (p *Executor) callQueryHistory(ctx context.Context, session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	if p.messageService == nil {
		return mcpgw.BuildToolErrorResult("history service not available"), nil
	}
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}

	hoursBack := intArg(arguments, "hours_back", 24)
	if hoursBack < 1 {
		hoursBack = 1
	}
	if hoursBack > maxHoursBack {
		hoursBack = maxHoursBack
	}

	limit := intArg(arguments, "limit", defaultLimit)
	if limit < 1 {
		limit = 1
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	roleFilter := strings.ToLower(strings.TrimSpace(mcpgw.FirstStringArg(arguments, "role")))
	keyword := strings.ToLower(strings.TrimSpace(mcpgw.FirstStringArg(arguments, "keyword")))

	since := time.Now().UTC().Add(-time.Duration(hoursBack) * time.Hour)
	msgs, err := p.messageService.ListSince(ctx, botID, since)
	if err != nil {
		p.logger.Warn("query_history failed", slog.Any("error", err))
		return mcpgw.BuildToolErrorResult("failed to query history: " + err.Error()), nil
	}

	type historyEntry struct {
		Role        string `json:"role"`
		Content     string `json:"content"`
		Sender      string `json:"sender,omitempty"`
		Platform    string `json:"platform,omitempty"`
		Time        string `json:"time"`
	}

	var results []historyEntry
	for _, msg := range msgs {
		if roleFilter != "" && msg.Role != roleFilter {
			continue
		}

		contentText := extractTextContent(msg.Content)
		if keyword != "" && !strings.Contains(strings.ToLower(contentText), keyword) {
			continue
		}

		sender := msg.SenderDisplayName
		if sender == "" && msg.Role == "user" {
			sender = "User"
		}

		results = append(results, historyEntry{
			Role:     msg.Role,
			Content:  truncateString(contentText, 500),
			Sender:   sender,
			Platform: msg.Platform,
			Time:     msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})

		if len(results) >= limit {
			break
		}
	}

	payload := map[string]any{
		"ok":         true,
		"count":      len(results),
		"hours_back": hoursBack,
		"messages":   results,
	}
	return mcpgw.BuildToolSuccessResult(payload), nil
}

func extractTextContent(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var obj struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if obj.Text != "" {
			return obj.Text
		}
		return obj.Content
	}
	return string(raw)
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func intArg(args map[string]any, key string, defaultVal int) int {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return defaultVal
	}
}
