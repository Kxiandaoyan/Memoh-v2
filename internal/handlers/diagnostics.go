package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
)

type DiagnosticsHandler struct {
	queries        *sqlc.Queries
	cfg            config.Config
	accountService *accounts.Service
	logger         *slog.Logger
}

func NewDiagnosticsHandler(log *slog.Logger, queries *sqlc.Queries, cfg config.Config, accountService *accounts.Service) *DiagnosticsHandler {
	return &DiagnosticsHandler{
		queries:        queries,
		cfg:            cfg,
		accountService: accountService,
		logger:         log.With(slog.String("handler", "diagnostics")),
	}
}

func (h *DiagnosticsHandler) Register(e *echo.Echo) {
	e.GET("/diagnostics", h.RunDiagnostics)
}

type DiagnosticCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "error", "warn"
	Message string `json:"message"`
	Latency int64  `json:"latency_ms"`
}

type DiagnosticsResponse struct {
	Checks    []DiagnosticCheck `json:"checks"`
	Overall   string            `json:"overall"` // "healthy", "degraded", "unhealthy"
	Timestamp string            `json:"timestamp"`
}

func (h *DiagnosticsHandler) RunDiagnostics(c echo.Context) error {
	if err := RequireAdmin(c, h.accountService); err != nil {
		return err
	}

	ctx := c.Request().Context()
	checks := []DiagnosticCheck{
		h.checkPostgreSQL(ctx),
		h.checkQdrant(ctx),
		h.checkAgentGateway(ctx),
		h.checkContainerd(),
		h.checkDiskSpace(),
	}

	overall := "healthy"
	for _, check := range checks {
		if check.Status == "error" {
			overall = "unhealthy"
			break
		}
		if check.Status == "warn" {
			overall = "degraded"
		}
	}

	return c.JSON(http.StatusOK, DiagnosticsResponse{
		Checks:    checks,
		Overall:   overall,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *DiagnosticsHandler) checkPostgreSQL(ctx context.Context) DiagnosticCheck {
	start := time.Now()
	_, err := h.queries.CountModels(ctx)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return DiagnosticCheck{
			Name:    "PostgreSQL",
			Status:  "error",
			Message: fmt.Sprintf("Query failed: %v", err),
			Latency: latency,
		}
	}
	return DiagnosticCheck{
		Name:    "PostgreSQL",
		Status:  "ok",
		Message: fmt.Sprintf("Connected to %s:%d/%s", h.cfg.Postgres.Host, h.cfg.Postgres.Port, h.cfg.Postgres.Database),
		Latency: latency,
	}
}

func (h *DiagnosticsHandler) checkQdrant(ctx context.Context) DiagnosticCheck {
	start := time.Now()
	url := h.cfg.Qdrant.BaseURL
	if url == "" {
		url = config.DefaultQdrantURL
	}
	healthURL := url + "/healthz"

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return DiagnosticCheck{
			Name:    "Qdrant",
			Status:  "error",
			Message: fmt.Sprintf("Request build failed: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return DiagnosticCheck{
			Name:    "Qdrant",
			Status:  "error",
			Message: fmt.Sprintf("Connection failed: %v", err),
			Latency: latency,
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return DiagnosticCheck{
			Name:    "Qdrant",
			Status:  "error",
			Message: fmt.Sprintf("Unhealthy (HTTP %d)", resp.StatusCode),
			Latency: latency,
		}
	}
	return DiagnosticCheck{
		Name:    "Qdrant",
		Status:  "ok",
		Message: fmt.Sprintf("Healthy at %s", url),
		Latency: latency,
	}
}

func (h *DiagnosticsHandler) checkAgentGateway(ctx context.Context) DiagnosticCheck {
	start := time.Now()
	gwURL := h.cfg.AgentGateway.BaseURL()
	healthURL := gwURL + "/"

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return DiagnosticCheck{
			Name:    "Agent Gateway",
			Status:  "error",
			Message: fmt.Sprintf("Request build failed: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return DiagnosticCheck{
			Name:    "Agent Gateway",
			Status:  "error",
			Message: fmt.Sprintf("Connection failed: %v", err),
			Latency: latency,
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return DiagnosticCheck{
			Name:    "Agent Gateway",
			Status:  "error",
			Message: fmt.Sprintf("Server error (HTTP %d)", resp.StatusCode),
			Latency: latency,
		}
	}
	return DiagnosticCheck{
		Name:    "Agent Gateway",
		Status:  "ok",
		Message: fmt.Sprintf("Reachable at %s", gwURL),
		Latency: latency,
	}
}

func (h *DiagnosticsHandler) checkContainerd() DiagnosticCheck {
	start := time.Now()
	socketPath := h.cfg.Containerd.SocketPath
	if socketPath == "" {
		socketPath = config.DefaultSocketPath
	}
	info, err := os.Stat(socketPath)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		if os.IsNotExist(err) {
			return DiagnosticCheck{
				Name:    "Containerd",
				Status:  "error",
				Message: fmt.Sprintf("Socket not found: %s", socketPath),
				Latency: latency,
			}
		}
		return DiagnosticCheck{
			Name:    "Containerd",
			Status:  "error",
			Message: fmt.Sprintf("Socket stat failed: %v", err),
			Latency: latency,
		}
	}
	if info.Mode()&os.ModeSocket == 0 {
		return DiagnosticCheck{
			Name:    "Containerd",
			Status:  "warn",
			Message: fmt.Sprintf("Path exists but is not a socket: %s", socketPath),
			Latency: latency,
		}
	}
	return DiagnosticCheck{
		Name:    "Containerd",
		Status:  "ok",
		Message: fmt.Sprintf("Socket available at %s", socketPath),
		Latency: latency,
	}
}

func (h *DiagnosticsHandler) checkDiskSpace() DiagnosticCheck {
	start := time.Now()
	dataRoot := h.cfg.MCP.DataRoot
	if dataRoot == "" {
		dataRoot = config.DefaultDataRoot
	}
	info, err := os.Stat(dataRoot)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return DiagnosticCheck{
			Name:    "Data Directory",
			Status:  "warn",
			Message: fmt.Sprintf("Data directory not found: %s", dataRoot),
			Latency: latency,
		}
	}
	if !info.IsDir() {
		return DiagnosticCheck{
			Name:    "Data Directory",
			Status:  "warn",
			Message: fmt.Sprintf("Path exists but is not a directory: %s", dataRoot),
			Latency: latency,
		}
	}
	return DiagnosticCheck{
		Name:    "Data Directory",
		Status:  "ok",
		Message: fmt.Sprintf("Available at %s", dataRoot),
		Latency: latency,
	}
}
