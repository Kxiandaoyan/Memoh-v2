package browser

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mcpcontainer "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/container"
)

// fakeExecRunner implements ExecRunner for testing.
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

// ─── Test Helpers ───────────────────────────────────────────────────────────

func newTestExecutor(runner mcpcontainer.ExecRunner) *Executor {
	return NewExecutor(nil, runner)
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

func TestExecutor_ListTools(t *testing.T) {
	runner := &fakeExecRunner{result: &mcpgw.ExecWithCaptureResult{}}
	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "test-bot"}

	tools, err := exec.ListTools(ctx, session)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all 14 tools are present
	expectedTools := map[string]bool{
		toolNavigate:         true,
		toolSnapshot:         true,
		toolClick:            true,
		toolFill:             true,
		toolGetText:          true,
		toolScreenshot:       true,
		toolGetURL:           true,
		toolStateSave:        true,
		toolStateLoad:        true,
		toolClose:            true,
		toolScroll:           true,
		toolWait:             true,
		toolActionbookSearch: true,
		toolActionbookGet:    true,
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("got %d tools, want %d", len(tools), len(expectedTools))
	}

	for _, tool := range tools {
		if !expectedTools[tool.Name] {
			t.Errorf("unexpected tool %q", tool.Name)
		}
		delete(expectedTools, tool.Name)
	}

	if len(expectedTools) > 0 {
		t.Errorf("missing tools: %v", expectedTools)
	}
}

// ─── browser_navigate Tests ─────────────────────────────────────────────────

func TestBrowserNavigate_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser navigate") {
				return nil, fmt.Errorf("expected agent-browser navigate command")
			}
			if !strings.Contains(cmd, "example.com") {
				return nil, fmt.Errorf("expected URL in command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"url":"https://example.com"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolNavigate, map[string]any{
		"url": "https://example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
	content := result["structuredContent"].(map[string]any)
	if content["success"] != true {
		t.Errorf("expected success=true")
	}
}

func TestBrowserNavigate_MissingURL(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolNavigate, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "url is required")
}

func TestBrowserNavigate_ExecFailure(t *testing.T) {
	runner := &fakeExecRunner{
		result: &mcpgw.ExecWithCaptureResult{
			Stderr:   "connection timeout",
			ExitCode: 1,
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolNavigate, map[string]any{
		"url": "https://timeout.example",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "navigate failed")
}

// ─── browser_snapshot Tests ─────────────────────────────────────────────────

func TestBrowserSnapshot_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser snapshot") {
				return nil, fmt.Errorf("expected snapshot command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout: `{"success":true,"elements":[
					{"ref":"@e1","tag":"button","text":"Submit","selector":"button.submit"},
					{"ref":"@e2","tag":"input","type":"text","selector":"input#username"}
				],"count":2}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolSnapshot, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
	content := result["structuredContent"].(map[string]any)
	if content["success"] != true {
		t.Errorf("expected success=true")
	}
	if content["count"].(float64) != 2 {
		t.Errorf("expected count=2, got %v", content["count"])
	}
}

// ─── browser_click Tests ────────────────────────────────────────────────────

func TestBrowserClick_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser click") {
				return nil, fmt.Errorf("expected click command")
			}
			if !strings.Contains(cmd, "@e1") {
				return nil, fmt.Errorf("expected selector in command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"selector":"@e1"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolClick, map[string]any{
		"selector": "@e1",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserClick_MissingSelector(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolClick, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "selector is required")
}

// ─── browser_fill Tests ─────────────────────────────────────────────────────

func TestBrowserFill_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser fill") {
				return nil, fmt.Errorf("expected fill command")
			}
			if !strings.Contains(cmd, "@e2") {
				return nil, fmt.Errorf("expected selector in command")
			}
			// Note: value might be shell-quoted
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"selector":"@e2","value":"test@example.com"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolFill, map[string]any{
		"selector": "@e2",
		"value":    "test@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserFill_MissingSelector(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolFill, map[string]any{
		"value": "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "selector is required")
}

// ─── browser_get_text Tests ─────────────────────────────────────────────────

func TestBrowserGetText_WithSelector(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser get-text") {
				return nil, fmt.Errorf("expected get-text command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"text":"Hello World"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolGetText, map[string]any{
		"selector": "@e1",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserGetText_FullPage(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser get-text") {
				return nil, fmt.Errorf("expected get-text command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"text":"Full page text content"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolGetText, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── browser_get_url Tests ──────────────────────────────────────────────────

func TestBrowserGetURL_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser get-url") {
				return nil, fmt.Errorf("expected get-url command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"url":"https://example.com/page"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolGetURL, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── browser_close Tests ────────────────────────────────────────────────────

func TestBrowserClose_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser close") {
				return nil, fmt.Errorf("expected close command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolClose, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── browser_scroll Tests ───────────────────────────────────────────────────

func TestBrowserScroll_WithCoordinates(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser scroll") {
				return nil, fmt.Errorf("expected scroll command")
			}
			if !strings.Contains(cmd, "--x") || !strings.Contains(cmd, "--y") {
				return nil, fmt.Errorf("expected x and y coordinates")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolScroll, map[string]any{
		"x": 100,
		"y": 200,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserScroll_WithSelector(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser scroll") {
				return nil, fmt.Errorf("expected scroll command")
			}
			if !strings.Contains(cmd, "@e1") {
				return nil, fmt.Errorf("expected selector in command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolScroll, map[string]any{
		"selector": "@e1",
		"y":        300,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── browser_wait Tests ─────────────────────────────────────────────────────

func TestBrowserWait_WithSelector(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser wait") {
				return nil, fmt.Errorf("expected wait command")
			}
			if !strings.Contains(cmd, "@e1") {
				return nil, fmt.Errorf("expected selector in command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWait, map[string]any{
		"selector": "@e1",
		"timeout":  5000,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserWait_TimeoutOnly(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser wait") {
				return nil, fmt.Errorf("expected wait command")
			}
			if !strings.Contains(cmd, "--timeout") {
				return nil, fmt.Errorf("expected timeout flag")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolWait, map[string]any{
		"timeout": 3000,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── browser_screenshot Tests ───────────────────────────────────────────────

func TestBrowserScreenshot_WithPath(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser screenshot") {
				return nil, fmt.Errorf("expected screenshot command")
			}
			if !strings.Contains(cmd, "--output") {
				return nil, fmt.Errorf("expected --output flag")
			}
			if !strings.Contains(cmd, "screenshot.png") {
				return nil, fmt.Errorf("expected path in command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"path":"screenshot.png"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolScreenshot, map[string]any{
		"path": "screenshot.png",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserScreenshot_WithoutPath(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if strings.Contains(cmd, "--output") {
				return nil, fmt.Errorf("should not have --output flag when path is empty")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"base64":"iVBORw0KGgo..."}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolScreenshot, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserScreenshot_FullPage(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "--full-page") {
				return nil, fmt.Errorf("expected --full-page flag")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolScreenshot, map[string]any{
		"full_page": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── browser_state_save/load Tests ──────────────────────────────────────────

func TestBrowserStateSave_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser state save") {
				return nil, fmt.Errorf("expected state save command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"path":"session.json"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolStateSave, map[string]any{
		"path": "session.json",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestBrowserStateLoad_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "agent-browser state load") {
				return nil, fmt.Errorf("expected state load command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout:   `{"success":true,"path":"session.json"}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolStateLoad, map[string]any{
		"path": "session.json",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── actionbook Tests ───────────────────────────────────────────────────────

func TestActionbookSearch_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "actionbook search") {
				return nil, fmt.Errorf("expected actionbook search command")
			}
			if !strings.Contains(cmd, "login") {
				return nil, fmt.Errorf("expected query in command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout: `{"success":true,"query":"login","results":[
					{"id":"github-login","name":"GitHub Login","score":0.95}
				],"count":1}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolActionbookSearch, map[string]any{
		"query": "login",
		"limit": 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

func TestActionbookGet_Success(t *testing.T) {
	runner := &fakeExecRunner{
		handler: func(req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error) {
			cmd := strings.Join(req.Command, " ")
			if !strings.Contains(cmd, "actionbook get") {
				return nil, fmt.Errorf("expected actionbook get command")
			}
			if !strings.Contains(cmd, "github-login") {
				return nil, fmt.Errorf("expected ID in command")
			}
			return &mcpgw.ExecWithCaptureResult{
				Stdout: `{"success":true,"id":"github-login","content":"..."}`,
				ExitCode: 0,
			}, nil
		},
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolActionbookGet, map[string]any{
		"id": "github-login",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoError(t, result)
}

// ─── Error Cases ────────────────────────────────────────────────────────────

func TestCallTool_NoBotID(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{} // Empty BotID

	result, err := exec.CallTool(ctx, session, toolNavigate, map[string]any{
		"url": "https://example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "bot_id is required")
}

func TestCallTool_ExecRunnerError(t *testing.T) {
	runner := &fakeExecRunner{
		err: fmt.Errorf("docker container not found"),
	}

	exec := newTestExecutor(runner)
	ctx := context.Background()
	session := mcpgw.ToolSessionContext{BotID: "bot1"}

	result, err := exec.CallTool(ctx, session, toolNavigate, map[string]any{
		"url": "https://example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertError(t, result, "navigate exec failed")
}

func TestCallTool_UnknownTool(t *testing.T) {
	runner := &fakeExecRunner{}
	exec := newTestExecutor(runner)
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
