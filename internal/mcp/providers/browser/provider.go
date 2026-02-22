package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mcpcontainer "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/container"
)

const (
	// Browser tools
	toolNavigate     = "browser_navigate"
	toolSnapshot     = "browser_snapshot"
	toolClick        = "browser_click"
	toolFill         = "browser_fill"
	toolGetText      = "browser_get_text"
	toolScreenshot   = "browser_screenshot"
	toolGetURL       = "browser_get_url"
	toolStateSave    = "browser_state_save"
	toolStateLoad    = "browser_state_load"
	toolClose        = "browser_close"
	toolScroll       = "browser_scroll"
	toolWait         = "browser_wait"

	// Actionbook tools
	toolActionbookSearch = "actionbook_search"
	toolActionbookGet    = "actionbook_get"

	execWorkDir = "/data"
)

// Executor provides browser automation tools backed by agent-browser CLI.
type Executor struct {
	logger     *slog.Logger
	execRunner mcpcontainer.ExecRunner
}

// NewExecutor creates a new browser tools executor.
func NewExecutor(log *slog.Logger, execRunner mcpcontainer.ExecRunner) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		logger:     log.With(slog.String("provider", "browser")),
		execRunner: execRunner,
	}
}

// ListTools returns all browser and actionbook tool descriptors.
func (e *Executor) ListTools(_ context.Context, _ mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolNavigate,
			Description: "Navigate browser to a URL. Opens a new browser session if none exists.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "URL to navigate to (e.g. https://example.com)",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        toolSnapshot,
			Description: "Capture interactive elements on current page. Returns elements with @e1, @e2, etc. references that can be used in click/fill commands.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        toolClick,
			Description: "Click an element on the page using @e reference from snapshot or CSS selector.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"selector": map[string]any{
						"type":        "string",
						"description": "Element reference like @e1 or CSS selector like button.submit",
					},
				},
				"required": []string{"selector"},
			},
		},
		{
			Name:        toolFill,
			Description: "Fill an input field with text using @e reference from snapshot or CSS selector.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"selector": map[string]any{
						"type":        "string",
						"description": "Element reference like @e1 or CSS selector like input[name='email']",
					},
					"value": map[string]any{
						"type":        "string",
						"description": "Text to fill into the input field",
					},
				},
				"required": []string{"selector", "value"},
			},
		},
		{
			Name:        toolGetText,
			Description: "Extract text content from an element or the entire page.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"selector": map[string]any{
						"type":        "string",
						"description": "Element reference like @e1, CSS selector, or empty for full page text",
					},
				},
			},
		},
		{
			Name:        toolScreenshot,
			Description: "Capture a screenshot of the current page. Returns base64-encoded image or saves to file.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "File path to save screenshot (optional, returns base64 if not provided)",
					},
					"full_page": map[string]any{
						"type":        "boolean",
						"description": "Capture full scrollable page instead of viewport only (default: false)",
					},
				},
			},
		},
		{
			Name:        toolGetURL,
			Description: "Get the current page URL.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        toolStateSave,
			Description: "Save current browser session state (cookies, localStorage, etc.) to file for later restoration.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "File path to save session state (e.g. session.json)",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        toolStateLoad,
			Description: "Restore browser session state from a previously saved file.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "File path to load session state from",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        toolClose,
			Description: "Close the browser session and cleanup resources.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        toolScroll,
			Description: "Scroll the page or a specific element.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"selector": map[string]any{
						"type":        "string",
						"description": "Element to scroll (optional, scrolls page if not provided)",
					},
					"x": map[string]any{
						"type":        "number",
						"description": "Horizontal scroll offset in pixels",
					},
					"y": map[string]any{
						"type":        "number",
						"description": "Vertical scroll offset in pixels",
					},
				},
			},
		},
		{
			Name:        toolWait,
			Description: "Wait for an element to appear on the page or for a specified duration.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"selector": map[string]any{
						"type":        "string",
						"description": "Element selector to wait for (optional)",
					},
					"timeout": map[string]any{
						"type":        "number",
						"description": "Maximum time to wait in milliseconds (default: 30000)",
					},
				},
			},
		},
		{
			Name:        toolActionbookSearch,
			Description: "Search for browser automation action sequences (actionbooks) in the library. Returns actionbook summaries matching the query.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query for finding actionbooks (e.g. 'login to github', 'search google')",
					},
					"limit": map[string]any{
						"type":        "number",
						"description": "Maximum number of results (default: 10)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        toolActionbookGet,
			Description: "Get detailed actionbook content by ID or name. Returns the full action sequence definition.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Actionbook ID or name to retrieve",
					},
				},
				"required": []string{"id"},
			},
		},
	}, nil
}

// CallTool dispatches to the appropriate browser or actionbook tool handler.
func (e *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}

	switch toolName {
	case toolNavigate:
		return e.navigate(ctx, botID, arguments)
	case toolSnapshot:
		return e.snapshot(ctx, botID)
	case toolClick:
		return e.click(ctx, botID, arguments)
	case toolFill:
		return e.fill(ctx, botID, arguments)
	case toolGetText:
		return e.getText(ctx, botID, arguments)
	case toolScreenshot:
		return e.screenshot(ctx, botID, arguments)
	case toolGetURL:
		return e.getURL(ctx, botID)
	case toolStateSave:
		return e.stateSave(ctx, botID, arguments)
	case toolStateLoad:
		return e.stateLoad(ctx, botID, arguments)
	case toolClose:
		return e.close(ctx, botID)
	case toolScroll:
		return e.scroll(ctx, botID, arguments)
	case toolWait:
		return e.wait(ctx, botID, arguments)
	case toolActionbookSearch:
		return e.actionbookSearch(ctx, botID, arguments)
	case toolActionbookGet:
		return e.actionbookGet(ctx, botID, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

// ─── Browser Tools ──────────────────────────────────────────────────────────

func (e *Executor) navigate(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	url := mcpgw.StringArg(args, "url")
	if url == "" {
		return mcpgw.BuildToolErrorResult("url is required"), nil
	}

	cmd := fmt.Sprintf("agent-browser navigate %s --json 2>&1", mcpcontainer.ShellQuote(url))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("navigate exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("navigate failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		// Fallback if not JSON
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"url":    url,
			"output": result.Stdout,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) snapshot(ctx context.Context, botID string) (map[string]any, error) {
	cmd := "agent-browser snapshot --json 2>&1"
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("snapshot exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("snapshot failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"elements": result.Stdout,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) click(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	selector := mcpgw.StringArg(args, "selector")
	if selector == "" {
		return mcpgw.BuildToolErrorResult("selector is required"), nil
	}

	cmd := fmt.Sprintf("agent-browser click %s --json 2>&1", mcpcontainer.ShellQuote(selector))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("click exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("click failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"selector": selector,
			"success":  true,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) fill(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	selector := mcpgw.StringArg(args, "selector")
	value := mcpgw.StringArg(args, "value")
	if selector == "" {
		return mcpgw.BuildToolErrorResult("selector is required"), nil
	}

	cmd := fmt.Sprintf("agent-browser fill %s %s --json 2>&1",
		mcpcontainer.ShellQuote(selector),
		mcpcontainer.ShellQuote(value))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("fill exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("fill failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"selector": selector,
			"value":    value,
			"success":  true,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) getText(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	selector := mcpgw.StringArg(args, "selector")

	cmd := "agent-browser get-text --json 2>&1"
	if selector != "" {
		cmd = fmt.Sprintf("agent-browser get-text %s --json 2>&1", mcpcontainer.ShellQuote(selector))
	}

	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("get-text exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("get-text failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"text": result.Stdout,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) screenshot(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	path := mcpgw.StringArg(args, "path")
	fullPage, _, _ := mcpgw.BoolArg(args, "full_page")

	cmdParts := []string{"agent-browser", "screenshot"}
	if path != "" {
		cmdParts = append(cmdParts, "--output", mcpcontainer.ShellQuote(path))
	}
	if fullPage {
		cmdParts = append(cmdParts, "--full-page")
	}
	cmdParts = append(cmdParts, "--json", "2>&1")

	cmd := strings.Join(cmdParts, " ")
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("screenshot exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("screenshot failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"path":   path,
			"output": result.Stdout,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) getURL(ctx context.Context, botID string) (map[string]any, error) {
	cmd := "agent-browser get-url --json 2>&1"
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("get-url exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("get-url failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"url": strings.TrimSpace(result.Stdout),
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) stateSave(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	path := mcpgw.StringArg(args, "path")
	if path == "" {
		return mcpgw.BuildToolErrorResult("path is required"), nil
	}

	cmd := fmt.Sprintf("agent-browser state save %s --json 2>&1", mcpcontainer.ShellQuote(path))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("state save exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("state save failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"path":    path,
			"success": true,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) stateLoad(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	path := mcpgw.StringArg(args, "path")
	if path == "" {
		return mcpgw.BuildToolErrorResult("path is required"), nil
	}

	cmd := fmt.Sprintf("agent-browser state load %s --json 2>&1", mcpcontainer.ShellQuote(path))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("state load exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("state load failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"path":    path,
			"success": true,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) close(ctx context.Context, botID string) (map[string]any, error) {
	cmd := "agent-browser close --json 2>&1"
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("close exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("close failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"success": true,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) scroll(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	selector := mcpgw.StringArg(args, "selector")
	x, hasX, _ := mcpgw.IntArg(args, "x")
	y, hasY, _ := mcpgw.IntArg(args, "y")

	cmdParts := []string{"agent-browser", "scroll"}
	if selector != "" {
		cmdParts = append(cmdParts, mcpcontainer.ShellQuote(selector))
	}
	if hasX {
		cmdParts = append(cmdParts, fmt.Sprintf("--x %d", x))
	}
	if hasY {
		cmdParts = append(cmdParts, fmt.Sprintf("--y %d", y))
	}
	cmdParts = append(cmdParts, "--json", "2>&1")

	cmd := strings.Join(cmdParts, " ")
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("scroll exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("scroll failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"success": true,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) wait(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	selector := mcpgw.StringArg(args, "selector")
	timeout, hasTimeout, _ := mcpgw.IntArg(args, "timeout")

	cmdParts := []string{"agent-browser", "wait"}
	if selector != "" {
		cmdParts = append(cmdParts, mcpcontainer.ShellQuote(selector))
	}
	if hasTimeout {
		cmdParts = append(cmdParts, fmt.Sprintf("--timeout %d", timeout))
	}
	cmdParts = append(cmdParts, "--json", "2>&1")

	cmd := strings.Join(cmdParts, " ")
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("wait exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("wait failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"success": true,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

// ─── Actionbook Tools ───────────────────────────────────────────────────────

func (e *Executor) actionbookSearch(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	query := mcpgw.StringArg(args, "query")
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}

	limit, hasLimit, _ := mcpgw.IntArg(args, "limit")
	if !hasLimit {
		limit = 10
	}

	cmd := fmt.Sprintf("actionbook search %s --limit %d --json 2>&1",
		mcpcontainer.ShellQuote(query), limit)
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("actionbook search exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("actionbook search failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		// Fallback if not JSON
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"query":   query,
			"results": result.Stdout,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}

func (e *Executor) actionbookGet(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	id := mcpgw.StringArg(args, "id")
	if id == "" {
		return mcpgw.BuildToolErrorResult("id is required"), nil
	}

	cmd := fmt.Sprintf("actionbook get %s --json 2>&1", mcpcontainer.ShellQuote(id))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("actionbook get exec failed: %s", err)), nil
	}
	if result.ExitCode != 0 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("actionbook get failed (exit %d): %s", result.ExitCode, result.Stderr)), nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		// Fallback if not JSON
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"id":      id,
			"content": result.Stdout,
		}), nil
	}
	return mcpgw.BuildToolSuccessResult(response), nil
}
