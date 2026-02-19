package processlog

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateProcessLogParams represents parameters for creating a process log
type CreateProcessLogParams struct {
	BotID      string
	ChatID     string
	TraceID    string
	UserID     string
	Channel    string
	Step       ProcessLogStep
	Level      ProcessLogLevel
	Message    string
	Data       map[string]any
	DurationMs int
}

// Service provides process log operations
type Service struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

// NewService creates a new process log service
func NewService(log *slog.Logger, queries *sqlc.Queries) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		queries: queries,
		logger:  log.With(slog.String("service", "processlog")),
	}
}

// Create creates a new process log entry
func (s *Service) Create(ctx context.Context, params CreateProcessLogParams) (*ProcessLog, error) {
	traceID := params.TraceID
	if traceID == "" {
		traceID = uuid.New().String()
	}

	level := params.Level
	if level == "" {
		level = LevelInfo
	}

	// Parse UUIDs
	botID, err := uuid.Parse(params.BotID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(params.ChatID)
	if err != nil {
		return nil, err
	}
	traceUUID, err := uuid.Parse(traceID)
	if err != nil {
		return nil, err
	}

	// Marshal data
	var dataJSON []byte
	if params.Data != nil {
		dataJSON, _ = json.Marshal(params.Data)
	} else {
		dataJSON = []byte("{}")
	}

	// Prepare nullable fields
	var userID, channel, message string
	if params.UserID != "" {
		userID = params.UserID
	}
	if params.Channel != "" {
		channel = params.Channel
	}
	if params.Message != "" {
		message = params.Message
	}

	dbParams := sqlc.CreateProcessLogParams{
		BotID:      pgtype.UUID{Bytes: botID, Valid: true},
		ChatID:     pgtype.UUID{Bytes: chatID, Valid: true},
		TraceID:    pgtype.UUID{Bytes: traceUUID, Valid: true},
		UserID:     pgtype.Text{String: userID, Valid: userID != ""},
		Channel:    pgtype.Text{String: channel, Valid: channel != ""},
		Step:       string(params.Step),
		Level:      string(level),
		Message:    pgtype.Text{String: message, Valid: message != ""},
		Data:       dataJSON,
		DurationMs: pgtype.Int4{Int32: int32(params.DurationMs), Valid: params.DurationMs > 0},
	}

	result, err := s.queries.CreateProcessLog(ctx, dbParams)
	if err != nil {
		s.logger.Warn("create process log failed", slog.Any("error", err))
		return nil, err
	}

	log := fromDBModel(result)
	return &log, nil
}

// GetRecentLogs retrieves recent process logs for a bot
func (s *Service) GetRecentLogs(ctx context.Context, botID string, limit int) ([]ProcessLog, error) {
	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 500 {
		limit = 500
	}

	rows, err := s.queries.GetRecentProcessLogs(ctx, sqlc.GetRecentProcessLogsParams{
		BotID: pgtype.UUID{Bytes: botUUID, Valid: true},
		Limit: int32(limit),
	})
	if err != nil {
		s.logger.Warn("get recent process logs failed", slog.Any("error", err))
		return nil, err
	}

	logs := make([]ProcessLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, fromDBModel(row))
	}

	return logs, nil
}

// GetLogsByTrace retrieves all logs for a specific trace
func (s *Service) GetLogsByTrace(ctx context.Context, traceID string) ([]ProcessLog, error) {
	traceUUID, err := uuid.Parse(traceID)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.GetProcessLogsByTrace(ctx, pgtype.UUID{Bytes: traceUUID, Valid: true})
	if err != nil {
		s.logger.Warn("get logs by trace failed", slog.Any("error", err))
		return nil, err
	}

	logs := make([]ProcessLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, fromDBModel(row))
	}

	return logs, nil
}

// GetLogsByChat retrieves logs for a specific chat
func (s *Service) GetLogsByChat(ctx context.Context, botID, chatID string, limit int) ([]ProcessLog, error) {
	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}
	chatUUID, err := uuid.Parse(chatID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 500 {
		limit = 500
	}

	rows, err := s.queries.GetProcessLogsByChat(ctx, sqlc.GetProcessLogsByChatParams{
		BotID:  pgtype.UUID{Bytes: botUUID, Valid: true},
		ChatID: pgtype.UUID{Bytes: chatUUID, Valid: true},
		Limit:  int32(limit),
	})
	if err != nil {
		s.logger.Warn("get logs by chat failed", slog.Any("error", err))
		return nil, err
	}

	logs := make([]ProcessLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, fromDBModel(row))
	}

	return logs, nil
}

// GetStats retrieves statistics for recent logs
func (s *Service) GetStats(ctx context.Context, botID string) ([]ProcessLogStats, error) {
	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.GetProcessLogStats(ctx, pgtype.UUID{Bytes: botUUID, Valid: true})
	if err != nil {
		s.logger.Warn("get process log stats failed", slog.Any("error", err))
		return nil, err
	}

	stats := make([]ProcessLogStats, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, ProcessLogStats{
			Step:          ProcessLogStep(row.Step.(string)),
			Count:         int(row.Count),
			AvgDurationMs: row.AvgDurationMs,
		})
	}

	return stats, nil
}

// ExportTrace builds a self-contained diagnostic report for a single trace.
func (s *Service) ExportTrace(ctx context.Context, traceID string) (*TraceExport, error) {
	logs, err := s.GetLogsByTrace(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}

	export := &TraceExport{
		Version:    "1.0",
		ExportedAt: time.Now().UTC(),
		TraceID:    traceID,
		BotID:      logs[0].BotID,
		ChatID:     logs[0].ChatID,
		Channel:    logs[0].Channel,
	}

	var summary TraceSummary
	var errors, warnings []string
	var totalDur int
	start := logs[0].CreatedAt
	end := logs[0].CreatedAt

	steps := make([]TraceExportStep, 0, len(logs))
	for _, l := range logs {
		if l.CreatedAt.Before(start) {
			start = l.CreatedAt
		}
		if l.CreatedAt.After(end) {
			end = l.CreatedAt
		}
		totalDur += l.DurationMs

		switch l.Step {
		case StepUserMessageReceived:
			if q, ok := l.Data["query"].(string); ok {
				summary.UserQuery = q
			}
		case StepPromptBuilt:
			if m, ok := l.Data["model"].(string); ok {
				summary.Model = m
			}
			if p, ok := l.Data["provider"].(string); ok {
				summary.Provider = p
			}
		case StepLLMResponseReceived:
			if u, ok := l.Data["usage"]; ok {
				if um, ok := u.(map[string]any); ok {
					summary.TokenUsage = um
				}
			}
			if rp, ok := l.Data["response_preview"].(string); ok && summary.AssistantResponse == "" {
				summary.AssistantResponse = rp
			}
		case StepResponseSent:
			if rp, ok := l.Data["response_preview"].(string); ok && summary.AssistantResponse == "" {
				summary.AssistantResponse = rp
			}
		}

		if l.Level == LevelError {
			errors = append(errors, l.Message)
		}
		if l.Level == LevelWarn {
			warnings = append(warnings, l.Message)
		}

		steps = append(steps, TraceExportStep{
			Step:       l.Step,
			Level:      l.Level,
			Message:    l.Message,
			Data:       l.Data,
			DurationMs: l.DurationMs,
			CreatedAt:  l.CreatedAt,
		})
	}

	summary.StepsCount = len(steps)
	summary.Errors = errors
	summary.Warnings = warnings

	export.TimeRange = TraceTimeRange{Start: start, End: end}
	export.TotalDurationMs = totalDur
	export.Summary = summary
	export.Steps = steps

	return export, nil
}

// CleanupOldLogs removes logs older than the specified duration
func (s *Service) CleanupOldLogs(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	err := s.queries.DeleteProcessLogsOlderThan(ctx, pgtype.Timestamptz{Time: cutoff})
	if err != nil {
		s.logger.Warn("cleanup old process logs failed", slog.Any("error", err))
		return 0, err
	}
	return 0, nil
}

// Helper functions

// fromDBModel converts from database model to domain model
func fromDBModel(l sqlc.ProcessLog) ProcessLog {
	return ProcessLog{
		ID:         l.ID.String(),
		BotID:      l.BotID.String(),
		ChatID:     l.ChatID.String(),
		TraceID:    l.TraceID.String(),
		UserID:     pgTextToString(l.UserID),
		Channel:    pgTextToString(l.Channel),
		Step:       ProcessLogStep(l.Step.(string)),
		Level:      ProcessLogLevel(l.Level.(string)),
		Message:    pgTextToString(l.Message),
		Data:       unmarshalJSON(l.Data),
		DurationMs: pgInt4ToInt(l.DurationMs),
		CreatedAt:  l.CreatedAt.Time,
	}
}

// pgTextToString converts pgtype.Text to string
func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// pgInt4ToInt converts pgtype.Int4 to int
func pgInt4ToInt(i pgtype.Int4) int {
	if !i.Valid {
		return 0
	}
	return int(i.Int32)
}

// unmarshalJSON unmarshals JSON data
func unmarshalJSON(data []byte) map[string]any {
	if len(data) == 0 || string(data) == "{}" {
		return make(map[string]any)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return make(map[string]any)
	}
	return result
}
