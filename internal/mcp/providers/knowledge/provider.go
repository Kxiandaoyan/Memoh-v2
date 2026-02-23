package knowledge

import (
	"context"
	"log/slog"
	"strings"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mem "github.com/Kxiandaoyan/Memoh-v2/internal/memory"
)

const (
	toolKnowledgeWrite    = "knowledge_write"
	toolKnowledgeRead     = "knowledge_read"
	knowledgeNamespace    = "knowledge"
	defaultKnowledgeLimit = 10
	maxKnowledgeLimit     = 50
)

// MemoryReadWriter is the subset of memory.Service needed by the knowledge provider.
type MemoryReadWriter interface {
	Add(ctx context.Context, req mem.AddRequest) (mem.SearchResponse, error)
	Search(ctx context.Context, req mem.SearchRequest) (mem.SearchResponse, error)
}

// Executor implements mcp.ToolExecutor for knowledge read/write.
type Executor struct {
	memory MemoryReadWriter
	logger *slog.Logger
}

func NewExecutor(log *slog.Logger, memory MemoryReadWriter) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		memory: memory,
		logger: log.With(slog.String("provider", "knowledge_tool")),
	}
}

func (e *Executor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	if e.memory == nil {
		return []mcpgw.ToolDescriptor{}, nil
	}
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolKnowledgeWrite,
			Description: "Write structured knowledge to the bot's knowledge base. Use this to persist facts, rules, or reference data the bot should remember long-term.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "The knowledge content to store",
					},
					"topic": map[string]any{
						"type":        "string",
						"description": "Optional topic tag for categorization",
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        toolKnowledgeRead,
			Description: "Search the bot's knowledge base for relevant entries.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query",
					},
					"topic": map[string]any{
						"type":        "string",
						"description": "Optional topic filter",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results (default 10)",
					},
				},
				"required": []string{"query"},
			},
		},
	}, nil
}

func (e *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if e.memory == nil {
		return mcpgw.BuildToolErrorResult("knowledge service not available"), nil
	}
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}

	switch toolName {
	case toolKnowledgeWrite:
		return e.handleWrite(ctx, botID, arguments)
	case toolKnowledgeRead:
		return e.handleRead(ctx, botID, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

func (e *Executor) handleWrite(ctx context.Context, botID string, arguments map[string]any) (map[string]any, error) {
	content := mcpgw.StringArg(arguments, "content")
	if content == "" {
		return mcpgw.BuildToolErrorResult("content is required"), nil
	}
	topic := mcpgw.StringArg(arguments, "topic")

	metadata := map[string]any{"source": "knowledge_tool"}
	if topic != "" {
		metadata["topic"] = topic
	}

	infer := false
	_, err := e.memory.Add(ctx, mem.AddRequest{
		Message:  content,
		BotID:    botID,
		Infer:    &infer,
		Metadata: metadata,
		Filters: map[string]any{
			"namespace": knowledgeNamespace,
			"scopeId":   botID,
			"bot_id":    botID,
		},
	})
	if err != nil {
		e.logger.Warn("knowledge write failed", slog.Any("error", err))
		return mcpgw.BuildToolErrorResult("knowledge write failed"), nil
	}

	return mcpgw.BuildToolSuccessResult(map[string]any{
		"status": "written",
		"topic":  topic,
	}), nil
}

func (e *Executor) handleRead(ctx context.Context, botID string, arguments map[string]any) (map[string]any, error) {
	query := mcpgw.StringArg(arguments, "query")
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}

	limit := defaultKnowledgeLimit
	if v, ok, err := mcpgw.IntArg(arguments, "limit"); err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	} else if ok && v > 0 {
		limit = v
	}
	if limit > maxKnowledgeLimit {
		limit = maxKnowledgeLimit
	}

	filters := map[string]any{
		"namespace": knowledgeNamespace,
		"scopeId":   botID,
		"bot_id":    botID,
	}
	if topic := mcpgw.StringArg(arguments, "topic"); topic != "" {
		filters["topic"] = topic
	}

	resp, err := e.memory.Search(ctx, mem.SearchRequest{
		Query:   query,
		BotID:   botID,
		Limit:   limit,
		Filters: filters,
		NoStats: true,
	})
	if err != nil {
		e.logger.Warn("knowledge read failed", slog.Any("error", err))
		return mcpgw.BuildToolErrorResult("knowledge read failed"), nil
	}

	results := make([]map[string]any, 0, len(resp.Results))
	for _, item := range resp.Results {
		results = append(results, map[string]any{
			"id":     item.ID,
			"memory": item.Memory,
			"score":  item.Score,
		})
	}

	return mcpgw.BuildToolSuccessResult(map[string]any{
		"query":   query,
		"total":   len(results),
		"results": results,
	}), nil
}
