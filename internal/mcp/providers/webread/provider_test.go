package webread

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mcpcontainer "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/container"
)

// ─── Mock Executors ─────────────────────────────────────────────────────────

// fakeExecRunner implements ExecRunner for testing curl/markdown strategy.
type fakeExecRunner struct {
	result  *mcpgw.ExecWithCaptureResult
	err     error
	lastReq mcpgw.ExecRequest
	handler func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error)
}

func (f *fakeExecRunner) ExecWithCapture(ctx context.Context, req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
	f.lastReq = req
	if f.handler != nil {
		return f.handler(req)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// fakeWebExecutor implements web provider interface for testing web_search strategy.
type fakeWebExecutor struct {
	handler func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error)
}

func (f *fakeWebExecutor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{Name: "web_search", Description: "Search the web"},
	}, nil
}

func (f *fakeWebExecutor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if f.handler != nil {
		return f.handler(ctx, session, toolName, arguments)
	}
	return mcpgw.BuildToolErrorResult("fake web executor not configured"), nil
}

// fakeBrowserExecutor implements browser provider interface for testing browser strategy.
type fakeBrowserExecutor struct {
	handler func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error)
}

func (f *fakeBrowserExecutor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{Name: "browser_navigate", Description: "Navigate to URL"},
		{Name: "browser_get_text", Description: "Get page text"},
		{Name: "browser_wait", Description: "Wait for element or timeout"},
		{Name: "actionbook_search", Description: "Search for actionbooks"},
		{Name: "actionbook_get", Description: "Get actionbook content"},
	}, nil
}

func (f *fakeBrowserExecutor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if f.handler != nil {
		return f.handler(ctx, session, toolName, arguments)
	}
	return mcpgw.BuildToolErrorResult("fake browser executor not configured"), nil
}

// ─── Test Helpers ───────────────────────────────────────────────────────────

func newTestExecutor(runner mcpcontainer.ExecRunner, webExec mcpgw.ToolExecutor, browserExec mcpgw.ToolExecutor) *Executor {
	return NewExecutor(nil, runner, webExec, browserExec)
}

func assertNoError(t *testing.T, result map[string]any) {
	t.Helper()
	if err := mcpgw.PayloadError(result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertError(t *testing.T, result map[string]any, expectedMsg string) {
	t.Helper()
	isErr, ok := result["isError"].(bool)
	if !ok || !isErr {
		t.Fatalf("expected error result with isError=true, got result: %+v", result)
	}
	content, ok := result["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content array in error result, got: %+v", result)
	}
	msg, ok := content[0]["text"].(string)
	if !ok {
		t.Fatalf("expected text field in content[0], got: %+v", content[0])
	}
	if !strings.Contains(msg, expectedMsg) {
		t.Errorf("error message %q does not contain %q", msg, expectedMsg)
	}
}

// ─── ListTools Tests ────────────────────────────────────────────────────────

func TestListTools(t *testing.T) {
	runner := &fakeExecRunner{result: &mcpgw.ExecWithCaptureResult{}}
	exec := newTestExecutor(runner, nil, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "test-bot"}

	tools, err := exec.ListTools(ctx, session)
	if err != nil {
		t.Fatal(err)
	}

	if len(tools) != 1 {
		t.Errorf("got %d tools, want 1", len(tools))
	}

	if tools[0].Name != toolWebRead {
		t.Errorf("got tool name %q, want %q", tools[0].Name, toolWebRead)
	}

	// Verify input schema
	schema := tools[0].InputSchema
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}

	if _, ok := props["url"]; !ok {
		t.Error("missing 'url' property in schema")
	}
	if _, ok := props["force_strategy"]; !ok {
		t.Error("missing 'force_strategy' property in schema")
	}
	if _, ok := props["include_metadata"]; !ok {
		t.Error("missing 'include_metadata' property in schema")
	}
}

// ─── web_read Tests - Level 1: Markdown Success ─────────────────────────────

func TestWebRead_MarkdownSuccess(t *testing.T) {
	markdownContent := `# Test Article

This is a test article with [links](https://example.com) and **formatting**.

## Section 1
Content here with proper markdown structure.

- List item 1
- List item 2

` + "```go\ncode block\n```"

	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "curl") {
				return nil, fmt.Errorf("expected curl command")
			}
			if !strings.Contains(cmd, "text/markdown") {
				return nil, fmt.Errorf("expected markdown accept header")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   markdownContent,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner, nil, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com/article",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	// Verify content
	structured := result["structuredContent"].(map[string]any)
	content, ok := structured["content"].(string)
	if !ok {
		t.Fatal("expected content string in response")
	}
	if !strings.Contains(content, "Test Article") {
		t.Error("content should contain markdown title")
	}

	// Verify metadata
	metadata, ok := structured["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata in response")
	}
	if metadata["final_method"] != "markdown" {
		t.Errorf("expected final_method=markdown, got %v", metadata["final_method"])
	}
	if metadata["success"] != true {
		t.Error("expected success=true")
	}

	// Verify attempts array
	attempts, ok := metadata["attempts"].([]string)
	if !ok {
		t.Fatal("expected attempts array in metadata")
	}
	if len(attempts) != 0 {
		t.Errorf("expected empty attempts for first-try success, got %v", attempts)
	}
}

// ─── web_read Tests - Level 2: Fallback to Web Search ───────────────────────

func TestWebRead_FallbackToWebSearch(t *testing.T) {
	htmlContent := `<!DOCTYPE html><html><body>Not markdown</body></html>`

	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			// Return HTML instead of markdown
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   htmlContent,
				ExitCode: 0,
			}, nil
		},
	}

	webExec := &fakeWebExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			if toolName != "web_search" {
				return mcpgw.BuildToolErrorResult("wrong tool"), nil
			}

			// Return successful search results
			return map[string]any{
				"isError": false,
				"structuredContent": map[string]any{
					"results": []any{
						map[string]any{
							"title":       "Example Article",
							"url":         "https://example.com/article",
							"description": "This is the article description from search results.",
						},
						map[string]any{
							"title":       "Related Article",
							"url":         "https://example.com/related",
							"description": "Another relevant result.",
						},
					},
				},
			}, nil
		},
	}

	exec := newTestExecutor(runner, webExec, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com/article",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	// Verify strategy
	structured := result["structuredContent"].(map[string]any)
	metadata := structured["metadata"].(map[string]any)
	if metadata["final_method"] != "web_search" {
		t.Errorf("expected final_method=web_search, got %v", metadata["final_method"])
	}

	// Verify attempts
	attempts := metadata["attempts"].([]string)
	if len(attempts) != 1 {
		t.Errorf("expected 1 failed attempt, got %d", len(attempts))
	}
	if attempts[0] != "markdown_header_failed" {
		t.Errorf("expected markdown_header_failed, got %v", attempts[0])
	}

	// Verify content contains search results
	content := structured["content"].(string)
	if !strings.Contains(content, "Search Results") {
		t.Error("content should contain search results header")
	}
	if !strings.Contains(content, "Example Article") {
		t.Error("content should contain first result title")
	}
}

// ─── web_read Tests - Level 4: Fallback to Browser ──────────────────────────

func TestWebRead_FallbackToBrowser(t *testing.T) {
	runner := &fakeExecRunner{
		err: fmt.Errorf("curl command failed"),
	}

	webExec := &fakeWebExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			// Simulate web_search failure
			return mcpgw.BuildToolErrorResult("search failed"), nil
		},
	}

	browserExec := &fakeBrowserExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			switch toolName {
			case "browser_navigate":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"success": true,
						"url":     args["url"],
					},
				}, nil
			case "browser_wait":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"success": true,
					},
				}, nil
			case "browser_get_text":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"success": true,
						"text":    "Page content extracted via browser automation",
					},
				}, nil
			default:
				return mcpgw.BuildToolErrorResult("unknown tool"), nil
			}
		},
	}

	exec := newTestExecutor(runner, webExec, browserExec)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com/article",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	// Verify strategy
	structured := result["structuredContent"].(map[string]any)
	metadata := structured["metadata"].(map[string]any)
	if metadata["final_method"] != "browser_automation" {
		t.Errorf("expected final_method=browser_automation, got %v", metadata["final_method"])
	}

	// Verify attempts (markdown failed → web search failed → actionbook failed → browser succeeded)
	attempts := metadata["attempts"].([]string)
	if len(attempts) != 3 {
		t.Errorf("expected 3 failed attempts, got %d: %v", len(attempts), attempts)
	}

	// Verify content
	content := structured["content"].(string)
	if !strings.Contains(content, "browser automation") {
		t.Error("content should be from browser extraction")
	}
}

// ─── web_read Tests - Force Strategy ─────────────────────────────────────────

func TestWebRead_ForceStrategy_Markdown(t *testing.T) {
	markdownContent := `# Forced Markdown

This content is fetched using forced markdown strategy.

- Item 1
- Item 2
`

	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   markdownContent,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner, nil, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com",
		"force_strategy":   "markdown",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	structured := result["structuredContent"].(map[string]any)
	metadata := structured["metadata"].(map[string]any)
	if metadata["final_method"] != "markdown" {
		t.Errorf("expected final_method=markdown, got %v", metadata["final_method"])
	}
}

func TestWebRead_ForceStrategy_Browser(t *testing.T) {
	browserExec := &fakeBrowserExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			switch toolName {
			case "browser_navigate":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"success": true,
					},
				}, nil
			case "browser_wait":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"success": true,
					},
				}, nil
			case "browser_get_text":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"text": "Forced browser content",
					},
				}, nil
			default:
				return mcpgw.BuildToolErrorResult("unknown tool"), nil
			}
		},
	}

	exec := newTestExecutor(&fakeExecRunner{}, nil, browserExec)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com",
		"force_strategy":   "browser",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	structured := result["structuredContent"].(map[string]any)
	metadata := structured["metadata"].(map[string]any)
	if metadata["final_method"] != "browser" {
		t.Errorf("expected final_method=browser, got %v", metadata["final_method"])
	}
}

func TestWebRead_ForceStrategy_UnsupportedStrategy(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner, nil, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":            "https://example.com",
		"force_strategy": "invalid_strategy",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "unsupported force_strategy")
}

func TestWebRead_ForceStrategy_BrowserUnavailable(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner, nil, nil) // No browser executor
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":            "https://example.com",
		"force_strategy": "browser",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "browser automation is not available")
}

// ─── web_read Tests - All Strategies Fail ────────────────────────────────────

func TestWebRead_AllStrategiesFail(t *testing.T) {
	runner := &fakeExecRunner{
		err: fmt.Errorf("exec failed"),
	}

	webExec := &fakeWebExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			return mcpgw.BuildToolErrorResult("web search failed"), nil
		},
	}

	browserExec := &fakeBrowserExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			return mcpgw.BuildToolErrorResult("browser failed"), nil
		},
	}

	exec := newTestExecutor(runner, webExec, browserExec)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "all strategies exhausted")
}

// ─── isValidMarkdown Tests ──────────────────────────────────────────────────

func TestIsValidMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "valid markdown with heading and link",
			content: `# Title

Content with [link](https://example.com) and more text to meet minimum length requirement.`,
			want: true,
		},
		{
			name: "valid markdown with list",
			content: `Some intro text here to meet length requirement.

- List item 1
- List item 2
- List item 3`,
			want: true,
		},
		{
			name: "valid markdown with code block",
			content: "Long enough content here.\n\n```go\nfunc main() {\n  fmt.Println(\"hello\")\n}\n```",
			want: true,
		},
		{
			name:    "invalid - HTML document",
			content: `<!DOCTYPE html><html><head><title>Test</title></head><body>Content</body></html>`,
			want:    false,
		},
		{
			name:    "invalid - contains <html> tag",
			content: `<html><body>Some content that is long enough but still HTML</body></html>`,
			want:    false,
		},
		{
			name:    "invalid - too short",
			content: `# Short`,
			want:    false,
		},
		{
			name:    "invalid - plain text no markdown",
			content: `This is just plain text content that is long enough but has no markdown indicators at all.`,
			want:    false,
		},
		{
			name: "valid - multiple markdown features",
			content: `# Main Title

## Subsection

Content with **bold** and *italic*.

- Item 1
- Item 2

[Link](https://example.com)

` + "```\ncode\n```",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidMarkdown(tt.content)
			if got != tt.want {
				t.Errorf("isValidMarkdown() = %v, want %v\nContent: %s", got, tt.want, tt.content)
			}
		})
	}
}

// ─── Error Cases ────────────────────────────────────────────────────────────

func TestWebRead_NoBotID(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner, nil, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{} // Empty BotID

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url": "https://example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "bot_id is required")
}

func TestWebRead_MissingURL(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner, nil, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "url is required")
}

func TestCallTool_UnknownTool(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner, nil, nil)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, "unknown_tool", map[string]any{})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if err != mcpgw.ErrToolNotFound {
		t.Errorf("expected ErrToolNotFound, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for unknown tool")
	}
}

// ─── web_read Tests - Level 3: Fallback to Actionbook ────────────────────────

func TestWebRead_FallbackToActionbook(t *testing.T) {
	htmlContent := `<!DOCTYPE html><html><body>Not markdown</body></html>`

	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			// Return HTML instead of markdown
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   htmlContent,
				ExitCode: 0,
			}, nil
		},
	}

	webExec := &fakeWebExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			// Simulate web_search failure
			return mcpgw.BuildToolErrorResult("search failed"), nil
		},
	}

	browserExec := &fakeBrowserExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			switch toolName {
			case "actionbook_search":
				// Return successful actionbook search
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"results": []any{
							map[string]any{
								"id":          "actionbook-123",
								"title":       "Example.com scraper",
								"description": "Scrapes content from example.com",
							},
						},
					},
				}, nil
			case "actionbook_get":
				// Return actionbook content
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"content": "Content extracted via actionbook automation sequence",
					},
				}, nil
			default:
				return mcpgw.BuildToolErrorResult("unknown tool"), nil
			}
		},
	}

	exec := newTestExecutor(runner, webExec, browserExec)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com/article",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	// Verify strategy
	structured := result["structuredContent"].(map[string]any)
	metadata := structured["metadata"].(map[string]any)
	if metadata["final_method"] != "actionbook" {
		t.Errorf("expected final_method=actionbook, got %v", metadata["final_method"])
	}

	// Verify attempts
	attempts := metadata["attempts"].([]string)
	if len(attempts) != 2 {
		t.Errorf("expected 2 failed attempts, got %d: %v", len(attempts), attempts)
	}
	if attempts[0] != "markdown_header_failed" {
		t.Errorf("expected markdown_header_failed, got %v", attempts[0])
	}
	if attempts[1] != "web_search_failed" {
		t.Errorf("expected web_search_failed, got %v", attempts[1])
	}

	// Verify content
	content := structured["content"].(string)
	if !strings.Contains(content, "actionbook") {
		t.Error("content should be from actionbook extraction")
	}
}

func TestWebRead_ActionbookNoResults(t *testing.T) {
	runner := &fakeExecRunner{
		err: fmt.Errorf("curl failed"),
	}

	webExec := &fakeWebExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			return mcpgw.BuildToolErrorResult("search failed"), nil
		},
	}

	browserExec := &fakeBrowserExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			switch toolName {
			case "actionbook_search":
				// Return empty results
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"results": []any{},
					},
				}, nil
			case "browser_navigate":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"success": true,
					},
				}, nil
			case "browser_wait":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"success": true,
					},
				}, nil
			case "browser_get_text":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"text": "Fallback to browser after actionbook failed",
					},
				}, nil
			default:
				return mcpgw.BuildToolErrorResult("unknown tool"), nil
			}
		},
	}

	exec := newTestExecutor(runner, webExec, browserExec)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com/article",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	// Should fallback to browser
	structured := result["structuredContent"].(map[string]any)
	metadata := structured["metadata"].(map[string]any)
	if metadata["final_method"] != "browser_automation" {
		t.Errorf("expected final_method=browser_automation, got %v", metadata["final_method"])
	}

	// Verify actionbook was attempted
	attempts := metadata["attempts"].([]string)
	actionbookAttempted := false
	for _, attempt := range attempts {
		if attempt == "actionbook_failed" {
			actionbookAttempted = true
			break
		}
	}
	if !actionbookAttempted {
		t.Error("expected actionbook to be attempted")
	}
}

func TestWebRead_ForceStrategy_Actionbook(t *testing.T) {
	browserExec := &fakeBrowserExecutor{
		handler: func(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, args map[string]any) (map[string]any, error) {
			switch toolName {
			case "actionbook_search":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"results": []any{
							map[string]any{
								"id":    "test-actionbook",
								"title": "Test Actionbook",
							},
						},
					},
				}, nil
			case "actionbook_get":
				return map[string]any{
					"isError": false,
					"structuredContent": map[string]any{
						"content": "Forced actionbook content",
					},
				}, nil
			default:
				return mcpgw.BuildToolErrorResult("unknown tool"), nil
			}
		},
	}

	exec := newTestExecutor(&fakeExecRunner{}, nil, browserExec)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":              "https://example.com",
		"force_strategy":   "actionbook",
		"include_metadata": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)

	structured := result["structuredContent"].(map[string]any)
	metadata := structured["metadata"].(map[string]any)
	if metadata["final_method"] != "actionbook" {
		t.Errorf("expected final_method=actionbook, got %v", metadata["final_method"])
	}

	content := structured["content"].(string)
	if !strings.Contains(content, "Forced actionbook") {
		t.Error("content should be from forced actionbook")
	}
}

func TestWebRead_ForceStrategy_ActionbookUnavailable(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner, nil, nil) // No browser executor
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWebRead, map[string]any{
		"url":            "https://example.com",
		"force_strategy": "actionbook",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "actionbook is not available")
}

