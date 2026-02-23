package openviking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Kxiandaoyan/Memoh-v2/internal/conversation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	dbsqlc "github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
)

const sessionExtractionTimeout = 60 * time.Second

// SessionExtractor implements flow.OVSessionExtractor. It commits conversations
// to OpenViking's session management for automatic long-term memory extraction.
type SessionExtractor struct {
	execRunner ExecRunner
	queries    *dbsqlc.Queries
	logger     *slog.Logger
}

func NewSessionExtractor(log *slog.Logger, execRunner ExecRunner, queries *dbsqlc.Queries) *SessionExtractor {
	if log == nil {
		log = slog.Default()
	}
	return &SessionExtractor{
		execRunner: execRunner,
		queries:    queries,
		logger:     log.With(slog.String("component", "ov_session_extractor")),
	}
}

func (s *SessionExtractor) isEnabled(ctx context.Context, botID string) bool {
	if s.queries == nil {
		s.logger.Debug("openviking.session.isEnabled: disabled", slog.String("reason", "queries is nil"))
		return false
	}
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		s.logger.Debug("openviking.session.isEnabled: disabled", slog.Any("reason", err))
		return false
	}
	row, err := s.queries.GetBotPrompts(ctx, botUUID)
	if err != nil {
		s.logger.Debug("openviking.session.isEnabled: disabled", slog.Any("reason", err))
		return false
	}
	if !row.EnableOpenviking {
		s.logger.Debug("openviking.session.isEnabled: disabled", slog.String("reason", "EnableOpenviking is false"))
		return false
	}
	return true
}

// ExtractSession commits the conversation messages to an OpenViking session,
// triggering automatic memory extraction. This is safe to call from a goroutine.
// Returns (output, error): empty output with nil error means extraction was skipped.
func (s *SessionExtractor) ExtractSession(ctx context.Context, botID, chatID string, messages []conversation.ModelMessage) (string, error) {
	if !s.isEnabled(ctx, botID) {
		return "", nil
	}
	if len(messages) == 0 {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(ctx, sessionExtractionTimeout)
	defer cancel()

	type simpleMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	simplified := make([]simpleMsg, 0, len(messages))
	for _, m := range messages {
		text := strings.TrimSpace(m.TextContent())
		if text == "" {
			continue
		}
		simplified = append(simplified, simpleMsg{
			Role:    m.Role,
			Content: text,
		})
	}
	if len(simplified) == 0 {
		return "", nil
	}

	msgJSON, err := json.Marshal(simplified)
	if err != nil {
		s.logger.Warn("failed to marshal messages for OV session", slog.Any("error", err))
		return "", err
	}

	sessionID := chatID
	if sessionID == "" {
		sessionID = botID
	}

	script := fmt.Sprintf(`import openviking as ov, json, os
from openviking.message import Part
if not os.path.isdir('%s'):
    print(json.dumps({"status": "skipped", "memories_extracted": 0}))
else:
    client = ov.SyncOpenViking(path='%s', config_file='%s')
    client.initialize()
    try:
        messages = json.loads(%s)
        session = client.session(%s)
        session.load()
        for msg in messages:
            role = msg.get("role", "user")
            content = msg.get("content", "")
            session.add_message(role, [Part.text(content)])
        result = session.commit()
        print(json.dumps({"status": "committed", "memories_extracted": result.get("memories_extracted", 0)}, default=str))
    finally:
        client.close()`,
		ovDataPath, ovDataPath, ovConfPath,
		pyStr(string(msgJSON)), pyStr(sessionID))

	result, err := s.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{shellCmd, shellFlag, fmt.Sprintf("python3 -c %s", shellQuote(script))},
		WorkDir: "/data",
	})
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no running task found") || strings.Contains(errMsg, "not found") {
			s.logger.Info("OV session extraction skipped: container not running",
				slog.String("bot_id", botID))
			return "", nil
		}
		s.logger.Warn("OV session extraction exec failed",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		return "", fmt.Errorf("exec failed: %w", err)
	}
	if result.ExitCode != 0 {
		errMsg := truncate(result.Stderr, 2000)
		s.logger.Warn("OV session extraction python error",
			slog.String("bot_id", botID),
			slog.Int("exit_code", int(result.ExitCode)),
			slog.String("stderr", errMsg))
		return "", fmt.Errorf("exit code %d: %s", result.ExitCode, errMsg)
	}
	output := truncate(strings.TrimSpace(result.Stdout), 300)
	s.logger.Info("OV session extraction completed",
		slog.String("bot_id", botID),
		slog.String("chat_id", chatID),
		slog.String("output", output))
	return output, nil
}
