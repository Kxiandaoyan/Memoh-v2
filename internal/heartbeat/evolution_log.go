package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
)

// EvolutionLog represents a single evolution execution record.
type EvolutionLog struct {
	ID                string                 `json:"id"`
	BotID             string                 `json:"bot_id"`
	HeartbeatConfigID string                 `json:"heartbeat_config_id,omitempty"`
	TriggerReason     string                 `json:"trigger_reason"`
	Status            string                 `json:"status"`
	ChangesSummary    string                 `json:"changes_summary,omitempty"`
	FilesModified     []string               `json:"files_modified,omitempty"`
	FilesSnapshot     map[string]string      `json:"files_snapshot,omitempty"`
	AgentResponse     string                 `json:"agent_response,omitempty"`
	StartedAt         time.Time              `json:"started_at"`
	CompletedAt       time.Time              `json:"completed_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
}

// CompleteEvolutionLogRequest is the payload for completing an evolution log.
type CompleteEvolutionLogRequest struct {
	Status         string   `json:"status"`
	ChangesSummary string   `json:"changes_summary,omitempty"`
	FilesModified  []string `json:"files_modified,omitempty"`
	AgentResponse  string   `json:"agent_response,omitempty"`
}

// ListEvolutionLogsResponse wraps a list of evolution logs.
type ListEvolutionLogsResponse struct {
	Items []EvolutionLog `json:"items"`
	Total int64          `json:"total"`
}

// ListEvolutionLogs returns evolution logs for a bot with pagination.
func (e *Engine) ListEvolutionLogs(ctx context.Context, botID string, limit, offset int) (ListEvolutionLogsResponse, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		e.logger.Warn("list evolution logs failed", slog.String("bot_id", botID), slog.Any("error", err))
		return ListEvolutionLogsResponse{}, err
	}
	rows, err := e.queries.ListEvolutionLogsByBot(ctx, sqlc.ListEvolutionLogsByBotParams{
		BotID:  pgBotID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		e.logger.Warn("list evolution logs failed", slog.String("bot_id", botID), slog.Any("error", err))
		return ListEvolutionLogsResponse{}, err
	}
	total, err := e.queries.CountEvolutionLogsByBot(ctx, pgBotID)
	if err != nil {
		e.logger.Warn("list evolution logs failed", slog.String("bot_id", botID), slog.Any("error", err))
		return ListEvolutionLogsResponse{}, err
	}
	items := make([]EvolutionLog, 0, len(rows))
	for _, row := range rows {
		items = append(items, toEvolutionLog(row))
	}
	// Enrich with files_snapshot (not covered by sqlc since it was added after codegen).
	if e.dbPool != nil && len(items) > 0 {
		e.enrichFilesSnapshot(ctx, items)
	}
	return ListEvolutionLogsResponse{Items: items, Total: total}, nil
}

// GetEvolutionLog returns a single evolution log by ID.
func (e *Engine) GetEvolutionLog(ctx context.Context, logID string) (EvolutionLog, error) {
	pgID, err := db.ParseUUID(logID)
	if err != nil {
		e.logger.Warn("get evolution log failed", slog.String("log_id", logID), slog.Any("error", err))
		return EvolutionLog{}, err
	}
	row, err := e.queries.GetEvolutionLog(ctx, pgID)
	if err != nil {
		e.logger.Warn("get evolution log failed", slog.String("log_id", logID), slog.Any("error", err))
		return EvolutionLog{}, err
	}
	return toEvolutionLog(row), nil
}

// CompleteEvolutionLog updates an evolution log with the result.
func (e *Engine) CompleteEvolutionLog(ctx context.Context, logID string, req CompleteEvolutionLogRequest) (EvolutionLog, error) {
	pgID, err := db.ParseUUID(logID)
	if err != nil {
		e.logger.Warn("complete evolution log failed", slog.String("log_id", logID), slog.Any("error", err))
		return EvolutionLog{}, err
	}
	validStatuses := map[string]bool{"completed": true, "failed": true, "skipped": true}
	if !validStatuses[req.Status] {
		return EvolutionLog{}, fmt.Errorf("invalid status: %s (must be completed, failed, or skipped)", req.Status)
	}
	row, err := e.queries.CompleteEvolutionLog(ctx, sqlc.CompleteEvolutionLogParams{
		ID:             pgID,
		Status:         req.Status,
		ChangesSummary: pgtype.Text{String: req.ChangesSummary, Valid: req.ChangesSummary != ""},
		FilesModified:  req.FilesModified,
		AgentResponse:  pgtype.Text{String: req.AgentResponse, Valid: req.AgentResponse != ""},
	})
	if err != nil {
		e.logger.Warn("complete evolution log failed", slog.String("log_id", logID), slog.Any("error", err))
		return EvolutionLog{}, err
	}
	return toEvolutionLog(row), nil
}

// enrichFilesSnapshot fetches files_snapshot for each item in-place via a
// direct DB query. Items with no snapshot are left unchanged.
func (e *Engine) enrichFilesSnapshot(ctx context.Context, items []EvolutionLog) {
	if e.dbPool == nil || len(items) == 0 {
		return
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	rows, err := e.dbPool.Query(ctx,
		`SELECT id::text, files_snapshot FROM evolution_logs WHERE id = ANY($1::uuid[]) AND files_snapshot IS NOT NULL`,
		ids,
	)
	if err != nil {
		e.logger.Warn("enrich files_snapshot: query failed", slog.Any("error", err))
		return
	}
	defer rows.Close()
	byID := make(map[string]map[string]string, len(items))
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			continue
		}
		var snap map[string]string
		if err := json.Unmarshal(raw, &snap); err == nil && len(snap) > 0 {
			byID[id] = snap
		}
	}
	for i := range items {
		if snap, ok := byID[items[i].ID]; ok {
			items[i].FilesSnapshot = snap
		}
	}
}

func toEvolutionLog(row sqlc.EvolutionLog) EvolutionLog {
	l := EvolutionLog{
		ID:            row.ID.String(),
		BotID:         row.BotID.String(),
		TriggerReason: row.TriggerReason,
		Status:        row.Status,
		FilesModified: row.FilesModified,
	}
	if row.HeartbeatConfigID.Valid {
		l.HeartbeatConfigID = row.HeartbeatConfigID.String()
	}
	if row.ChangesSummary.Valid {
		l.ChangesSummary = row.ChangesSummary.String
	}
	if row.AgentResponse.Valid {
		l.AgentResponse = row.AgentResponse.String
	}
	if row.StartedAt.Valid {
		l.StartedAt = row.StartedAt.Time
	}
	if row.CompletedAt.Valid {
		l.CompletedAt = row.CompletedAt.Time
	}
	if row.CreatedAt.Valid {
		l.CreatedAt = row.CreatedAt.Time
	}
	return l
}
