package openviking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	dbsqlc "github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
)

const (
	ovDataPath = "/app/openviking-data"
	ovConfPath = "/data/ov.conf"

	toolOVInitialize    = "ov_initialize"
	toolOVFind          = "ov_find"
	toolOVSearch        = "ov_search"
	toolOVRead          = "ov_read"
	toolOVAbstract      = "ov_abstract"
	toolOVOverview      = "ov_overview"
	toolOVLs            = "ov_ls"
	toolOVTree          = "ov_tree"
	toolOVAddResource   = "ov_add_resource"
	toolOVRm            = "ov_rm"
	toolOVSessionCommit = "ov_session_commit"

	shellCmd  = "/bin/sh"
	shellFlag = "-c"
)

var allOVTools = []string{
	toolOVInitialize, toolOVFind, toolOVSearch, toolOVRead,
	toolOVAbstract, toolOVOverview, toolOVLs, toolOVTree,
	toolOVAddResource, toolOVRm, toolOVSessionCommit,
}

type ExecRunner interface {
	ExecWithCapture(ctx context.Context, req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error)
}

type Executor struct {
	execRunner ExecRunner
	queries    *dbsqlc.Queries
	logger     *slog.Logger
}

func NewExecutor(log *slog.Logger, execRunner ExecRunner, queries *dbsqlc.Queries) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		execRunner: execRunner,
		queries:    queries,
		logger:     log.With(slog.String("provider", "openviking_tool")),
	}
}

func (e *Executor) isEnabled(ctx context.Context, botID string) bool {
	if e.queries == nil {
		return false
	}
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		return false
	}
	row, err := e.queries.GetBotPrompts(ctx, botUUID)
	if err != nil {
		return false
	}
	return row.EnableOpenviking
}

func (e *Executor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	if e.execRunner == nil {
		return []mcpgw.ToolDescriptor{}, nil
	}
	botID := strings.TrimSpace(session.BotID)
	if botID == "" || !e.isEnabled(ctx, botID) {
		return []mcpgw.ToolDescriptor{}, nil
	}
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolOVInitialize,
			Description: "Initialize OpenViking context database for this bot. Call once before using other ov_* tools.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        toolOVFind,
			Description: "Semantic search across OpenViking context. Returns matching URIs with relevance scores.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
					"target_uri": map[string]any{
						"type":        "string",
						"description": "Scope search to a specific viking:// URI (e.g. viking://resources/). Defaults to all.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum results (default 10)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        toolOVSearch,
			Description: "Advanced retrieval with intent analysis and hierarchical directory-recursive search.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
					"target_uri": map[string]any{
						"type":        "string",
						"description": "Scope search to a specific viking:// URI. Defaults to all.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum results (default 10)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        toolOVRead,
			Description: "Read full content (L2) from a viking:// URI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"uri": map[string]any{
						"type":        "string",
						"description": "Viking URI (e.g. viking://resources/my_project/docs/api.md)",
					},
				},
				"required": []string{"uri"},
			},
		},
		{
			Name:        toolOVAbstract,
			Description: "Get L0 abstract (~100 tokens one-sentence summary) of a viking:// URI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"uri": map[string]any{
						"type":        "string",
						"description": "Viking URI",
					},
				},
				"required": []string{"uri"},
			},
		},
		{
			Name:        toolOVOverview,
			Description: "Get L1 overview (~2k tokens summary with structure and key points) of a viking:// URI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"uri": map[string]any{
						"type":        "string",
						"description": "Viking URI",
					},
				},
				"required": []string{"uri"},
			},
		},
		{
			Name:        toolOVLs,
			Description: "List directory contents under a viking:// URI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"uri": map[string]any{
						"type":        "string",
						"description": "Viking URI directory (e.g. viking://resources/)",
					},
					"recursive": map[string]any{
						"type":        "boolean",
						"description": "List subdirectories recursively",
					},
				},
				"required": []string{"uri"},
			},
		},
		{
			Name:        toolOVTree,
			Description: "Get a tree view of a viking:// directory structure.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"uri": map[string]any{
						"type":        "string",
						"description": "Viking URI directory",
					},
				},
				"required": []string{"uri"},
			},
		},
		{
			Name:        toolOVAddResource,
			Description: "Add a resource (URL, file path, or directory) to OpenViking. The resource will be parsed, vectorized, and indexed.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "URL or local file/directory path to add",
					},
					"wait": map[string]any{
						"type":        "boolean",
						"description": "Wait for processing to complete before returning (default false)",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        toolOVRm,
			Description: "Remove a resource from OpenViking by its viking:// URI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"uri": map[string]any{
						"type":        "string",
						"description": "Viking URI to remove",
					},
					"recursive": map[string]any{
						"type":        "boolean",
						"description": "Remove recursively (for directories)",
					},
				},
				"required": []string{"uri"},
			},
		},
		{
			Name:        toolOVSessionCommit,
			Description: "Commit the current conversation to OpenViking session, archiving messages and extracting long-term memories.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{
						"type":        "string",
						"description": "Session ID (auto-generated if empty)",
					},
					"messages": map[string]any{
						"type":        "string",
						"description": "JSON array of {role, content} message objects to commit",
					},
				},
				"required": []string{"messages"},
			},
		},
	}, nil
}

func (e *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	found := false
	for _, t := range allOVTools {
		if t == toolName {
			found = true
			break
		}
	}
	if !found {
		return nil, mcpgw.ErrToolNotFound
	}

	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}
	if !e.isEnabled(ctx, botID) {
		return mcpgw.BuildToolErrorResult("OpenViking is not enabled for this bot"), nil
	}

	if toolName != toolOVInitialize {
		e.ensureInitialized(ctx, botID)
	}

	switch toolName {
	case toolOVInitialize:
		return e.callInitialize(ctx, botID)
	case toolOVFind:
		return e.callFind(ctx, botID, arguments)
	case toolOVSearch:
		return e.callSearch(ctx, botID, arguments)
	case toolOVRead:
		return e.callRead(ctx, botID, arguments)
	case toolOVAbstract:
		return e.callAbstract(ctx, botID, arguments)
	case toolOVOverview:
		return e.callOverview(ctx, botID, arguments)
	case toolOVLs:
		return e.callLs(ctx, botID, arguments)
	case toolOVTree:
		return e.callTree(ctx, botID, arguments)
	case toolOVAddResource:
		return e.callAddResource(ctx, botID, arguments)
	case toolOVRm:
		return e.callRm(ctx, botID, arguments)
	case toolOVSessionCommit:
		return e.callSessionCommit(ctx, botID, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

// ensureInitialized lazily initializes the OpenViking data directory inside
// the container if it hasn't been set up yet. A lightweight check runs first;
// the full initialize only fires when the sentinel file is absent.
func (e *Executor) ensureInitialized(ctx context.Context, botID string) {
	check, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{shellCmd, shellFlag, fmt.Sprintf("test -d %s && echo ok", ovDataPath)},
		WorkDir: "/data",
	})
	if err == nil && strings.TrimSpace(check.Stdout) == "ok" {
		return
	}
	e.logger.Info("auto-initializing OpenViking", slog.String("bot_id", botID))
	if _, err := e.callInitialize(ctx, botID); err != nil {
		e.logger.Warn("auto-initialize failed", slog.String("bot_id", botID), slog.Any("error", err))
	}
}

// InitializeBot can be called externally (e.g. when OpenViking is first
// enabled in the prompts handler) to pre-initialize the data directory.
func (e *Executor) InitializeBot(ctx context.Context, botID string) {
	e.ensureInitialized(ctx, botID)
}

func (e *Executor) callInitialize(ctx context.Context, botID string) (map[string]any, error) {
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
client.close()
print(json.dumps({"status": "initialized"}))`, ovDataPath, ovConfPath)
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callFind(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	query := mcpgw.StringArg(args, "query")
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}
	targetURI := mcpgw.StringArg(args, "target_uri")
	limit := 10
	if v, ok, _ := mcpgw.IntArg(args, "limit"); ok && v > 0 {
		limit = v
	}
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    results = client.find(%s, target_uri=%s, limit=%d)
    items = [{"uri": r.uri, "score": round(r.score, 4)} for r in results.resources]
    print(json.dumps({"results": items, "total": len(items)}))
finally:
    client.close()`,
		ovDataPath, ovConfPath,
		pyStr(query), pyStr(targetURI), limit)
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callSearch(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	query := mcpgw.StringArg(args, "query")
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}
	targetURI := mcpgw.StringArg(args, "target_uri")
	limit := 10
	if v, ok, _ := mcpgw.IntArg(args, "limit"); ok && v > 0 {
		limit = v
	}
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    results = client.search(%s, target_uri=%s, limit=%d)
    items = [{"uri": r.uri, "score": round(r.score, 4)} for r in results.resources]
    print(json.dumps({"results": items, "total": len(items)}))
finally:
    client.close()`,
		ovDataPath, ovConfPath,
		pyStr(query), pyStr(targetURI), limit)
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callRead(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	uri := mcpgw.StringArg(args, "uri")
	if uri == "" {
		return mcpgw.BuildToolErrorResult("uri is required"), nil
	}
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    content = client.read(%s)
    print(json.dumps({"uri": %s, "content": content}))
finally:
    client.close()`,
		ovDataPath, ovConfPath, pyStr(uri), pyStr(uri))
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callAbstract(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	uri := mcpgw.StringArg(args, "uri")
	if uri == "" {
		return mcpgw.BuildToolErrorResult("uri is required"), nil
	}
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    text = client.abstract(%s)
    print(json.dumps({"uri": %s, "abstract": text}))
finally:
    client.close()`,
		ovDataPath, ovConfPath, pyStr(uri), pyStr(uri))
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callOverview(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	uri := mcpgw.StringArg(args, "uri")
	if uri == "" {
		return mcpgw.BuildToolErrorResult("uri is required"), nil
	}
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    text = client.overview(%s)
    print(json.dumps({"uri": %s, "overview": text}))
finally:
    client.close()`,
		ovDataPath, ovConfPath, pyStr(uri), pyStr(uri))
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callLs(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	uri := mcpgw.StringArg(args, "uri")
	if uri == "" {
		return mcpgw.BuildToolErrorResult("uri is required"), nil
	}
	recursive, _, _ := mcpgw.BoolArg(args, "recursive")
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    entries = client.ls(%s, recursive=%s)
    print(json.dumps({"uri": %s, "entries": entries}))
finally:
    client.close()`,
		ovDataPath, ovConfPath, pyStr(uri), pyBool(recursive), pyStr(uri))
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callTree(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	uri := mcpgw.StringArg(args, "uri")
	if uri == "" {
		return mcpgw.BuildToolErrorResult("uri is required"), nil
	}
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    tree = client.tree(%s)
    print(json.dumps({"uri": %s, "tree": tree}))
finally:
    client.close()`,
		ovDataPath, ovConfPath, pyStr(uri), pyStr(uri))
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callAddResource(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	path := mcpgw.StringArg(args, "path")
	if path == "" {
		return mcpgw.BuildToolErrorResult("path is required"), nil
	}
	wait, _, _ := mcpgw.BoolArg(args, "wait")
	waitLine := ""
	if wait {
		waitLine = "\n    client.wait_processed()"
	}
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    result = client.add_resource(path=%s)%s
    print(json.dumps(result, default=str))
finally:
    client.close()`,
		ovDataPath, ovConfPath, pyStr(path), waitLine)
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callRm(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	uri := mcpgw.StringArg(args, "uri")
	if uri == "" {
		return mcpgw.BuildToolErrorResult("uri is required"), nil
	}
	recursive, _, _ := mcpgw.BoolArg(args, "recursive")
	script := fmt.Sprintf(`import openviking as ov, json
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    client.rm(%s, recursive=%s)
    print(json.dumps({"status": "removed", "uri": %s}))
finally:
    client.close()`,
		ovDataPath, ovConfPath, pyStr(uri), pyBool(recursive), pyStr(uri))
	return e.runPython(ctx, botID, script)
}

func (e *Executor) callSessionCommit(ctx context.Context, botID string, args map[string]any) (map[string]any, error) {
	messagesRaw := mcpgw.StringArg(args, "messages")
	if messagesRaw == "" {
		return mcpgw.BuildToolErrorResult("messages is required"), nil
	}
	sessionID := mcpgw.StringArg(args, "session_id")

	sessionInit := "import uuid; sid = str(uuid.uuid4())"
	if sessionID != "" {
		sessionInit = fmt.Sprintf("sid = %s", pyStr(sessionID))
	}
	script := fmt.Sprintf(`import openviking as ov, json
from openviking.message import Part
%s
client = ov.SyncOpenViking(path='%s', config_file='%s')
client.initialize()
try:
    messages = json.loads(%s)
    session = client.session(sid)
    session.load()
    for msg in messages:
        role = msg.get("role", "user")
        content = msg.get("content", "")
        session.add_message(role, [Part.text(content)])
    result = session.commit()
    print(json.dumps(result, default=str))
finally:
    client.close()`,
		sessionInit, ovDataPath, ovConfPath,
		pyStr(messagesRaw))
	return e.runPython(ctx, botID, script)
}

// runPython executes a Python script inside the bot container and parses stdout as JSON.
func (e *Executor) runPython(ctx context.Context, botID, script string) (map[string]any, error) {
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{shellCmd, shellFlag, fmt.Sprintf("python3 -c %s", shellQuote(script))},
		WorkDir: "/data",
	})
	if err != nil {
		e.logger.Warn("openviking exec failed",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("container exec failed: %v", err)), nil
	}
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(result.Stderr)
		if errMsg == "" {
			errMsg = strings.TrimSpace(result.Stdout)
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("python exited with code %d", result.ExitCode)
		}
		e.logger.Warn("openviking python error",
			slog.String("bot_id", botID),
			slog.Int("exit_code", int(result.ExitCode)),
			slog.String("stderr", truncate(result.Stderr, 500)))
		return mcpgw.BuildToolErrorResult(errMsg), nil
	}

	stdout := strings.TrimSpace(result.Stdout)
	if stdout == "" {
		return mcpgw.BuildToolSuccessResult(map[string]any{"status": "ok"}), nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		return mcpgw.BuildToolSuccessResult(map[string]any{"output": stdout}), nil
	}
	return mcpgw.BuildToolSuccessResult(parsed), nil
}

// pyStr produces a Python string literal with proper escaping.
func pyStr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return "'" + s + "'"
}

func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// shellQuote wraps a string for safe use as a single shell argument.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
