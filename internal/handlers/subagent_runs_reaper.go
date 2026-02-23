package handlers

import (
	"context"
	"log/slog"
	"time"
)

// StartStaleRunReaper periodically aborts subagent runs stuck in "running"
// longer than maxAge. It blocks until ctx is cancelled.
func (h *SubagentRunsHandler) StartStaleRunReaper(ctx context.Context, maxAge time.Duration) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	h.logger.Info("stale run reaper started", slog.Duration("max_age", maxAge))
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("stale run reaper stopped")
			return
		case <-ticker.C:
			h.reapStaleRuns(ctx, maxAge)
		}
	}
}

func (h *SubagentRunsHandler) reapStaleRuns(ctx context.Context, maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	tag, err := h.pool.Exec(ctx,
		`UPDATE subagent_runs
		 SET status='aborted', error_message='timed out', ended_at=now()
		 WHERE status='running' AND started_at < $1`, cutoff)
	if err != nil {
		h.logger.Warn("reaper query failed", slog.Any("error", err))
		return
	}
	if tag.RowsAffected() > 0 {
		h.logger.Info("reaped stale runs", slog.Int64("aborted", tag.RowsAffected()))
	}
}
