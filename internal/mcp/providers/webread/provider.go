package webread

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mcpcontainer "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/container"
)

const (
	toolWebRead = "web_read"

	execWorkDir = "/data"
)

// Executor provides a smart web content retrieval tool with multi-level degradation:
// Level 1: Try markdown header (Accept: text/markdown)
// Level 2: Try web_search (if available)
// Level 3: Try actionbook (pre-defined automation sequences)
// Level 4: Fallback to browser automation
type Executor struct {
	logger      *slog.Logger
	execRunner  mcpcontainer.ExecRunner
	webExec     mcpgw.ToolExecutor
	browserExec mcpgw.ToolExecutor
}

// NewExecutor creates a new webread executor with all necessary dependencies.
// webExec and browserExec may be nil, but this will disable certain degradation levels.
func NewExecutor(log *slog.Logger, execRunner mcpcontainer.ExecRunner, webExec mcpgw.ToolExecutor, browserExec mcpgw.ToolExecutor) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		logger:      log.With(slog.String("provider", "webread")),
		execRunner:  execRunner,
		webExec:     webExec,
		browserExec: browserExec,
	}
}

// ListTools returns the web_read tool descriptor.
func (e *Executor) ListTools(_ context.Context, _ mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolWebRead,
			Description: "Intelligently fetch web content using multi-level degradation strategy (markdown → search → actionbook → browser). Returns clean text content.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "URL to fetch content from (e.g. https://example.com/article)",
					},
					"force_strategy": map[string]any{
						"type":        "string",
						"description": "Force a specific strategy: 'markdown', 'search', 'actionbook', 'browser' (optional, auto-detect by default)",
					},
					"include_metadata": map[string]any{
						"type":        "boolean",
						"description": "Include degradation metadata (attempts, final method, timing) in response (default: false)",
					},
				},
				"required": []string{"url"},
			},
		},
	}, nil
}

// CallTool dispatches to the webRead handler.
func (e *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}

	switch toolName {
	case toolWebRead:
		return e.webRead(ctx, session, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

// webRead implements the main multi-level degradation logic.
func (e *Executor) webRead(ctx context.Context, session mcpgw.ToolSessionContext, args map[string]any) (map[string]any, error) {
	url := strings.TrimSpace(mcpgw.StringArg(args, "url"))
	if url == "" {
		return mcpgw.BuildToolErrorResult("url is required"), nil
	}

	forceStrategy := strings.ToLower(strings.TrimSpace(mcpgw.StringArg(args, "force_strategy")))
	includeMetadata, _, _ := mcpgw.BoolArg(args, "include_metadata")

	metadata := newDegradationMetadata()
	botID := session.BotID

	// If force_strategy is specified, skip degradation
	if forceStrategy != "" {
		return e.executeForceStrategy(ctx, botID, session, url, forceStrategy, includeMetadata, metadata)
	}

	// Level 1: Try Markdown Header
	e.logger.Info("attempting markdown header strategy", slog.String("url", url))
	if result, err := e.tryMarkdownHeader(ctx, botID, url); err == nil {
		metadata.FinalMethod = "markdown"
		metadata.Complete()
		e.logger.Info("markdown header succeeded", slog.String("url", url))
		return buildResponse(result, metadata, includeMetadata), nil
	}
	metadata.AddAttempt("markdown_header_failed")
	e.logger.Debug("markdown header failed, trying next strategy", slog.String("url", url))

	// Level 2: Try Web Search (if webExec is available)
	if e.webExec != nil {
		e.logger.Info("attempting web search strategy", slog.String("url", url))
		if result, err := e.tryWebSearch(ctx, session, url); err == nil {
			metadata.FinalMethod = "web_search"
			metadata.Complete()
			e.logger.Info("web search succeeded", slog.String("url", url))
			return buildResponse(result, metadata, includeMetadata), nil
		}
		metadata.AddAttempt("web_search_failed")
		e.logger.Debug("web search failed, trying next strategy", slog.String("url", url))
	}

	// Level 3: Try Actionbook (if browserExec is available)
	if e.browserExec != nil {
		e.logger.Info("attempting actionbook strategy", slog.String("url", url))
		if result, err := e.tryActionbook(ctx, session, url); err == nil {
			metadata.FinalMethod = "actionbook"
			metadata.Complete()
			e.logger.Info("actionbook succeeded", slog.String("url", url))
			return buildResponse(result, metadata, includeMetadata), nil
		}
		metadata.AddAttempt("actionbook_failed")
		e.logger.Debug("actionbook failed, trying next strategy", slog.String("url", url))
	}

	// Level 4: Fallback to Browser Automation
	if e.browserExec != nil {
		e.logger.Info("attempting browser automation strategy", slog.String("url", url))
		if result, err := e.tryAgentBrowser(ctx, session, url); err == nil {
			metadata.FinalMethod = "browser_automation"
			metadata.Complete()
			e.logger.Info("browser automation succeeded", slog.String("url", url))
			return buildResponse(result, metadata, includeMetadata), nil
		}
		metadata.AddAttempt("browser_automation_failed")
		e.logger.Error("all strategies failed", slog.String("url", url))
	}

	// All strategies failed
	metadata.FinalMethod = "all_failed"
	metadata.Complete()
	return mcpgw.BuildToolErrorResult(fmt.Sprintf("failed to fetch content from %s: all strategies exhausted", url)), nil
}

// executeForceStrategy handles forced strategy execution (skip degradation).
func (e *Executor) executeForceStrategy(
	ctx context.Context,
	botID string,
	session mcpgw.ToolSessionContext,
	url string,
	strategy string,
	includeMetadata bool,
	metadata *degradationMetadata,
) (map[string]any, error) {
	metadata.FinalMethod = strategy

	var result string
	var err error

	switch strategy {
	case "markdown":
		result, err = e.tryMarkdownHeader(ctx, botID, url)
	case "search":
		if e.webExec == nil {
			return mcpgw.BuildToolErrorResult("web_search is not available (webExec not initialized)"), nil
		}
		result, err = e.tryWebSearch(ctx, session, url)
	case "actionbook":
		if e.browserExec == nil {
			return mcpgw.BuildToolErrorResult("actionbook is not available (browserExec not initialized)"), nil
		}
		result, err = e.tryActionbook(ctx, session, url)
	case "browser":
		if e.browserExec == nil {
			return mcpgw.BuildToolErrorResult("browser automation is not available (browserExec not initialized)"), nil
		}
		result, err = e.tryAgentBrowser(ctx, session, url)
	default:
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("unsupported force_strategy: %s (supported: markdown, search, actionbook, browser)", strategy)), nil
	}

	metadata.Complete()

	if err != nil {
		metadata.AddAttempt(fmt.Sprintf("%s_failed", strategy))
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("forced strategy '%s' failed: %s", strategy, err.Error())), nil
	}

	return buildResponse(result, metadata, includeMetadata), nil
}

// buildResponse constructs the final tool response with optional metadata.
func buildResponse(content string, metadata *degradationMetadata, includeMetadata bool) map[string]any {
	data := map[string]any{
		"content": content,
	}

	if includeMetadata {
		data["metadata"] = map[string]any{
			"final_method":  metadata.FinalMethod,
			"attempts":      metadata.Attempts,
			"duration_ms":   metadata.Duration.Milliseconds(),
			"success":       metadata.FinalMethod != "all_failed",
		}
	}

	return mcpgw.BuildToolSuccessResult(data)
}
