package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/processlog"
)

// ProcessLogHandler handles process log operations
type ProcessLogHandler struct {
	botService *bots.Service
	service    *processlog.Service
	logger     *slog.Logger
}

// NewProcessLogHandler creates a new ProcessLogHandler
func NewProcessLogHandler(botService *bots.Service, service *processlog.Service, log *slog.Logger) *ProcessLogHandler {
	if log == nil {
		log = slog.Default()
	}
	return &ProcessLogHandler{
		botService: botService,
		service:    service,
		logger:     log.With(slog.String("handler", "processlog")),
	}
}

// Register registers process log routes
func (h *ProcessLogHandler) Register(g *echo.Group) {
	g.GET("/logs/recent", h.GetRecentLogs)
	g.GET("/logs/trace/:traceId", h.GetLogsByTrace)
	g.GET("/logs/chat/:chatId", h.GetLogsByChat)
	g.GET("/logs/stats", h.GetStats)
}

// GetRecentLogs returns recent process logs for the current user's bots
func (h *ProcessLogHandler) GetRecentLogs(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current user ID from context
	userID := extractUserID(c)
	if userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	// Get user's bots
	bots, err := h.botService.ListByOwner(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to list user bots", slog.Any("error", err))
		return err
	}

	if len(bots) == 0 {
		return c.JSON(http.StatusOK, []processlog.ProcessLog{})
	}

	// Get recent logs from the first bot (simplified - could aggregate from all)
	botID := bots[0].ID
	limit := 500
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := h.service.GetRecentLogs(ctx, botID, limit)
	if err != nil {
		h.logger.Warn("failed to get recent logs", slog.Any("error", err))
		return err
	}

	return c.JSON(http.StatusOK, logs)
}

// GetLogsByTrace returns all logs for a specific trace
func (h *ProcessLogHandler) GetLogsByTrace(c echo.Context) error {
	ctx := c.Request().Context()
	traceID := c.Param("traceId")

	logs, err := h.service.GetLogsByTrace(ctx, traceID)
	if err != nil {
		h.logger.Warn("failed to get logs by trace", slog.Any("error", err))
		return err
	}

	return c.JSON(http.StatusOK, logs)
}

// GetLogsByChat returns logs for a specific chat
func (h *ProcessLogHandler) GetLogsByChat(c echo.Context) error {
	ctx := c.Request().Context()
	chatID := c.Param("chatId")

	// Get bot ID from query param
	botID := c.QueryParam("botId")
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "botId is required")
	}

	limit := 100
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := h.service.GetLogsByChat(ctx, botID, chatID, limit)
	if err != nil {
		h.logger.Warn("failed to get logs by chat", slog.Any("error", err))
		return err
	}

	return c.JSON(http.StatusOK, logs)
}

// GetStats returns statistics for recent logs
func (h *ProcessLogHandler) GetStats(c echo.Context) error {
	ctx := c.Request().Context()

	userID := extractUserID(c)
	if userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	bots, err := h.botService.ListByOwner(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to list user bots", slog.Any("error", err))
		return err
	}

	if len(bots) == 0 {
		return c.JSON(http.StatusOK, []processlog.ProcessLogStats{})
	}

	botID := bots[0].ID
	stats, err := h.service.GetStats(ctx, botID)
	if err != nil {
		h.logger.Warn("failed to get stats", slog.Any("error", err))
		return err
	}

	return c.JSON(http.StatusOK, stats)
}

// Helper function to extract user ID from context
func extractUserID(c echo.Context) string {
	// Try to get from JWT claims
	if claims := extractUserFromToken(c); claims != nil {
		if claims.Subject != "" {
			return claims.Subject
		}
	}
	// Try to get from header (for testing)
	if userID := c.Request().Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	return ""
}

type tokenClaims struct {
	Subject string `json:"sub"`
	Exp     int64  `json:"exp"`
	Iat     int64  `json:"iat"`
}

func extractUserFromToken(c echo.Context) *tokenClaims {
	// This is a simplified version - the actual implementation depends on your auth setup
	return nil
}

// ProcessLogEntry is a helper to create process log entries
type ProcessLogEntry struct {
	BotID      string
	ChatID     string
	TraceID    string
	UserID     string
	Channel    string
	Step       processlog.ProcessLogStep
	Level      processlog.ProcessLogLevel
	Message    string
	Data       map[string]any
	DurationMs int
	Service    *processlog.Service
	Logger     *slog.Logger
}

// Log creates a process log entry
func (p *ProcessLogEntry) Log(ctx context.Context) error {
	if p.Service == nil {
		return nil
	}

	req := processlog.CreateProcessLogParams{
		BotID:      p.BotID,
		ChatID:     p.ChatID,
		TraceID:    p.TraceID,
		UserID:     p.UserID,
		Channel:    p.Channel,
		Step:       p.Step,
		Level:      p.Level,
		Message:    p.Message,
		Data:       p.Data,
		DurationMs: p.DurationMs,
	}

	_, err := p.Service.Create(ctx, req)
	if err != nil && p.Logger != nil {
		p.Logger.Warn("failed to create process log", slog.Any("error", err))
	}
	return err
}

// LogDuration logs with duration measurement
func (p *ProcessLogEntry) LogDuration(ctx context.Context, startTime time.Time) {
	p.DurationMs = int(time.Since(startTime).Milliseconds())
	_ = p.Log(ctx)
}
