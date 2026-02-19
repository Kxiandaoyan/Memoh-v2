package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/auth"
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
func (h *ProcessLogHandler) Register(e *echo.Echo) {
	e.GET("/logs/recent", h.GetRecentLogs)
	e.GET("/logs/trace/:traceId", h.GetLogsByTrace)
	e.GET("/logs/trace/:traceId/export", h.ExportTrace)
	e.GET("/logs/chat/:chatId", h.GetLogsByChat)
	e.GET("/logs/chat/:chatId/export", h.ExportChat)
	e.GET("/logs/stats", h.GetStats)
}

// GetRecentLogs returns recent process logs for the current user's bots
func (h *ProcessLogHandler) GetRecentLogs(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		return err
	}

	bots, err := h.botService.ListByOwner(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to list user bots", slog.Any("error", err))
		return err
	}

	if len(bots) == 0 {
		return c.JSON(http.StatusOK, []processlog.ProcessLog{})
	}

	limit := 500
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	botID := c.QueryParam("botId")
	if botID != "" {
		logs, err := h.service.GetRecentLogs(ctx, botID, limit)
		if err != nil {
			h.logger.Warn("failed to get recent logs", slog.Any("error", err))
			return err
		}
		return c.JSON(http.StatusOK, logs)
	}

	var allLogs []processlog.ProcessLog
	for _, bot := range bots {
		logs, err := h.service.GetRecentLogs(ctx, bot.ID, limit)
		if err != nil {
			h.logger.Warn("failed to get recent logs for bot", slog.String("bot_id", bot.ID), slog.Any("error", err))
			continue
		}
		allLogs = append(allLogs, logs...)
	}
	return c.JSON(http.StatusOK, allLogs)
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

// ExportTrace returns a structured diagnostic report for a single trace
func (h *ProcessLogHandler) ExportTrace(c echo.Context) error {
	ctx := c.Request().Context()
	traceID := c.Param("traceId")

	export, err := h.service.ExportTrace(ctx, traceID)
	if err != nil {
		h.logger.Warn("failed to export trace", slog.Any("error", err))
		return err
	}
	if export == nil {
		return echo.NewHTTPError(http.StatusNotFound, "trace not found")
	}

	return c.JSON(http.StatusOK, export)
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

// ExportChat returns a multi-round diagnostic report for an entire chat session
func (h *ProcessLogHandler) ExportChat(c echo.Context) error {
	ctx := c.Request().Context()
	chatID := c.Param("chatId")
	botID := c.QueryParam("botId")
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "botId is required")
	}

	limit := 2000
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	export, err := h.service.ExportChat(ctx, botID, chatID, limit)
	if err != nil {
		h.logger.Warn("failed to export chat", slog.Any("error", err))
		return err
	}
	if export == nil {
		return echo.NewHTTPError(http.StatusNotFound, "no logs found for this chat")
	}

	return c.JSON(http.StatusOK, export)
}

// GetStats returns statistics for recent logs
func (h *ProcessLogHandler) GetStats(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		return err
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

