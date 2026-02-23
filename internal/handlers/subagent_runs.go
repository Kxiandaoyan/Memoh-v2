package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// SubagentRunsHandler persists sub-agent run state from the Agent Gateway so
// runs survive restarts and are queryable via the Web UI.
type SubagentRunsHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewSubagentRunsHandler creates a new handler backed by the given pool.
func NewSubagentRunsHandler(pool *pgxpool.Pool, log *slog.Logger) *SubagentRunsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &SubagentRunsHandler{
		pool:   pool,
		logger: log.With(slog.String("handler", "subagent_runs")),
	}
}

func (h *SubagentRunsHandler) Register(e *echo.Echo) {
	e.POST("/subagent-runs", h.Create)
	e.PATCH("/subagent-runs/:runId", h.Update)
	e.GET("/subagent-runs", h.List)
	e.GET("/subagent-runs/:runId", h.Get)
	e.DELETE("/subagent-runs/:runId", h.Delete)
}

// subagentRunRow is the database representation of a sub-agent run.
type subagentRunRow struct {
	ID            string     `json:"id"`
	RunID         string     `json:"run_id"`
	BotID         string     `json:"bot_id"`
	Name          string     `json:"name"`
	Task          string     `json:"task"`
	Status        string     `json:"status"`
	SpawnDepth    int        `json:"spawn_depth"`
	ParentRunID   *string    `json:"parent_run_id,omitempty"`
	ResultSummary *string    `json:"result_summary,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type createSubagentRunRequest struct {
	RunID        string  `json:"run_id"`
	BotID        string  `json:"bot_id"`
	Name         string  `json:"name"`
	Task         string  `json:"task"`
	SpawnDepth   int     `json:"spawn_depth"`
	ParentRunID  *string `json:"parent_run_id,omitempty"`
}

type updateSubagentRunRequest struct {
	Status        string  `json:"status"`
	ResultSummary *string `json:"result_summary,omitempty"`
	ErrorMessage  *string `json:"error_message,omitempty"`
}

// Create registers a new sub-agent run.
func (h *SubagentRunsHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	var req createSubagentRunRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.RunID == "" || req.BotID == "" || req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "run_id, bot_id, and name are required")
	}

	_, err := h.pool.Exec(ctx,
		`INSERT INTO subagent_runs (run_id, bot_id, name, task, status, spawn_depth, parent_run_id)
		 VALUES ($1, $2, $3, $4, 'running', $5, $6)
		 ON CONFLICT (run_id) DO NOTHING`,
		req.RunID, req.BotID, req.Name, req.Task, req.SpawnDepth, req.ParentRunID,
	)
	if err != nil {
		h.logger.Warn("create subagent run failed", slog.String("run_id", req.RunID), slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create subagent run")
	}
	return c.JSON(http.StatusCreated, map[string]string{"run_id": req.RunID, "status": "running"})
}

// Update changes the status of an existing run.
func (h *SubagentRunsHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	runID := c.Param("runId")
	if runID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "runId is required")
	}
	var req updateSubagentRunRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "status is required")
	}

	var endedAt *time.Time
	if req.Status == "completed" || req.Status == "failed" || req.Status == "aborted" {
		now := time.Now()
		endedAt = &now
	}

	_, err := h.pool.Exec(ctx,
		`UPDATE subagent_runs
		 SET status=$1, result_summary=$2, error_message=$3, ended_at=$4
		 WHERE run_id=$5`,
		req.Status, req.ResultSummary, req.ErrorMessage, endedAt, runID,
	)
	if err != nil {
		h.logger.Warn("update subagent run failed", slog.String("run_id", runID), slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update subagent run")
	}
	return c.JSON(http.StatusOK, map[string]string{"run_id": runID, "status": req.Status})
}

// List returns sub-agent runs, optionally filtered by bot_id and status.
func (h *SubagentRunsHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	botID := c.QueryParam("botId")
	status := c.QueryParam("status")

	query := `SELECT id, run_id, bot_id, name, task, status, spawn_depth,
	                 parent_run_id, result_summary, error_message,
	                 started_at, ended_at, created_at
	          FROM subagent_runs WHERE 1=1`
	args := []any{}
	idx := 1

	if botID != "" {
		query += fmt.Sprintf(` AND bot_id=$%d`, idx)
		args = append(args, botID)
		idx++
	}
	if status != "" {
		query += fmt.Sprintf(` AND status=$%d`, idx)
		args = append(args, status)
		idx++
	}
	_ = idx // suppress unused variable warning
	query += ` ORDER BY created_at DESC LIMIT 200`

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list subagent runs")
	}
	defer rows.Close()

	results := make([]subagentRunRow, 0)
	for rows.Next() {
		var r subagentRunRow
		if err := rows.Scan(&r.ID, &r.RunID, &r.BotID, &r.Name, &r.Task, &r.Status,
			&r.SpawnDepth, &r.ParentRunID, &r.ResultSummary, &r.ErrorMessage,
			&r.StartedAt, &r.EndedAt, &r.CreatedAt); err != nil {
			h.logger.Warn("scan subagent run row failed", slog.Any("error", err))
			continue
		}
		results = append(results, r)
	}
	return c.JSON(http.StatusOK, results)
}

// Get returns a single sub-agent run by run_id.
func (h *SubagentRunsHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	runID := c.Param("runId")
	if runID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "runId is required")
	}
	var r subagentRunRow
	err := h.pool.QueryRow(ctx,
		`SELECT id, run_id, bot_id, name, task, status, spawn_depth,
		        parent_run_id, result_summary, error_message,
		        started_at, ended_at, created_at
		 FROM subagent_runs WHERE run_id=$1`, runID,
	).Scan(&r.ID, &r.RunID, &r.BotID, &r.Name, &r.Task, &r.Status,
		&r.SpawnDepth, &r.ParentRunID, &r.ResultSummary, &r.ErrorMessage,
		&r.StartedAt, &r.EndedAt, &r.CreatedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found")
	}
	return c.JSON(http.StatusOK, r)
}

// Delete removes a run record.
func (h *SubagentRunsHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	runID := c.Param("runId")
	if runID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "runId is required")
	}
	_, err := h.pool.Exec(ctx, `DELETE FROM subagent_runs WHERE run_id=$1`, runID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete subagent run")
	}
	return c.JSON(http.StatusOK, map[string]string{"run_id": runID, "deleted": "true"})
}
