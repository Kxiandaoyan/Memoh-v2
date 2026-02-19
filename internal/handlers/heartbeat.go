package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/heartbeat"
)

// rollbackEvolutionResult is the response body for RollbackEvolutionLog.
type rollbackEvolutionResult struct {
	LogID          string   `json:"log_id"`
	FilesRestored  []string `json:"files_restored"`
}

// HeartbeatHandler handles HTTP requests for heartbeat configuration.
type HeartbeatHandler struct {
	engine         *heartbeat.Engine
	botService     *bots.Service
	accountService *accounts.Service
	logger         *slog.Logger
}

// NewHeartbeatHandler creates a new HeartbeatHandler.
func NewHeartbeatHandler(log *slog.Logger, engine *heartbeat.Engine, botService *bots.Service, accountService *accounts.Service) *HeartbeatHandler {
	return &HeartbeatHandler{
		engine:         engine,
		botService:     botService,
		accountService: accountService,
		logger:         log.With(slog.String("handler", "heartbeat")),
	}
}

// Register registers the heartbeat routes.
func (h *HeartbeatHandler) Register(e *echo.Echo) {
	group := e.Group("/bots/:bot_id/heartbeat")
	group.POST("", h.Create)
	group.GET("", h.List)
	group.GET("/:id", h.Get)
	group.PUT("/:id", h.Update)
	group.DELETE("/:id", h.Delete)
	group.POST("/:id/trigger", h.Trigger)

	// Evolution log routes
	evoGroup := e.Group("/bots/:bot_id/evolution-logs")
	evoGroup.GET("", h.ListEvolutionLogs)
	evoGroup.GET("/:id", h.GetEvolutionLog)
	evoGroup.POST("/:id/complete", h.CompleteEvolutionLog)
	evoGroup.POST("/:id/rollback", h.RollbackEvolutionLog)
}

// Create godoc
// @Summary Create heartbeat config
// @Description Create a heartbeat configuration for a bot
// @Tags heartbeat
// @Param payload body heartbeat.CreateRequest true "Heartbeat config payload"
// @Success 201 {object} heartbeat.Config
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/heartbeat [post]
func (h *HeartbeatHandler) Create(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	var req heartbeat.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	resp, err := h.engine.Create(c.Request().Context(), botID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, resp)
}

// List godoc
// @Summary List heartbeat configs
// @Description List heartbeat configurations for a bot
// @Tags heartbeat
// @Success 200 {object} heartbeat.ListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/heartbeat [get]
func (h *HeartbeatHandler) List(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	items, err := h.engine.List(c.Request().Context(), botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, heartbeat.ListResponse{Items: items})
}

// Get godoc
// @Summary Get heartbeat config
// @Description Get a heartbeat configuration by ID
// @Tags heartbeat
// @Param id path string true "Heartbeat config ID"
// @Success 200 {object} heartbeat.Config
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/heartbeat/{id} [get]
func (h *HeartbeatHandler) Get(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	item, err := h.engine.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if item.BotID != botID {
		return echo.NewHTTPError(http.StatusForbidden, "bot mismatch")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, item)
}

// Update godoc
// @Summary Update heartbeat config
// @Description Update a heartbeat configuration by ID
// @Tags heartbeat
// @Param id path string true "Heartbeat config ID"
// @Param payload body heartbeat.UpdateRequest true "Heartbeat config payload"
// @Success 200 {object} heartbeat.Config
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/heartbeat/{id} [put]
func (h *HeartbeatHandler) Update(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	var req heartbeat.UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	item, err := h.engine.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if item.BotID != botID {
		return echo.NewHTTPError(http.StatusForbidden, "bot mismatch")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	resp, err := h.engine.Update(c.Request().Context(), id, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

// Delete godoc
// @Summary Delete heartbeat config
// @Description Delete a heartbeat configuration by ID
// @Tags heartbeat
// @Param id path string true "Heartbeat config ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/heartbeat/{id} [delete]
func (h *HeartbeatHandler) Delete(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	item, err := h.engine.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if item.BotID != botID {
		return echo.NewHTTPError(http.StatusForbidden, "bot mismatch")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	if err := h.engine.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// Trigger godoc
// @Summary Trigger heartbeat manually
// @Description Manually trigger a heartbeat configuration to fire immediately
// @Tags heartbeat
// @Param bot_id path string true "Bot ID"
// @Param id path string true "Heartbeat config ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/heartbeat/{id}/trigger [post]
func (h *HeartbeatHandler) Trigger(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	item, err := h.engine.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if item.BotID != botID {
		return echo.NewHTTPError(http.StatusForbidden, "bot mismatch")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	if err := h.engine.Fire(c.Request().Context(), id, "manual"); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "triggered"})
}

func (h *HeartbeatHandler) requireUserID(c echo.Context) (string, error) {
	return RequireChannelIdentityID(c)
}

func (h *HeartbeatHandler) authorizeBotAccess(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error) {
	return AuthorizeBotAccess(ctx, h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: false})
}

// ── Evolution Log Endpoints ─────────────────────────────────────────

// ListEvolutionLogs godoc
// @Summary List evolution logs
// @Description List evolution log entries for a bot with pagination
// @Tags evolution
// @Param bot_id path string true "Bot ID"
// @Param limit query int false "Max items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} heartbeat.ListEvolutionLogsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/evolution-logs [get]
func (h *HeartbeatHandler) ListEvolutionLogs(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	limit := 20
	if v := c.QueryParam("limit"); v != "" {
		if parsed, pErr := strconv.Atoi(v); pErr == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	offset := 0
	if v := c.QueryParam("offset"); v != "" {
		if parsed, pErr := strconv.Atoi(v); pErr == nil && parsed >= 0 {
			offset = parsed
		}
	}
	resp, err := h.engine.ListEvolutionLogs(c.Request().Context(), botID, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

// GetEvolutionLog godoc
// @Summary Get evolution log
// @Description Get a single evolution log entry by ID
// @Tags evolution
// @Param bot_id path string true "Bot ID"
// @Param id path string true "Evolution log ID"
// @Success 200 {object} heartbeat.EvolutionLog
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /bots/{bot_id}/evolution-logs/{id} [get]
func (h *HeartbeatHandler) GetEvolutionLog(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	logID := c.Param("id")
	if logID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	item, err := h.engine.GetEvolutionLog(c.Request().Context(), logID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if item.BotID != botID {
		return echo.NewHTTPError(http.StatusForbidden, "bot mismatch")
	}
	return c.JSON(http.StatusOK, item)
}

// CompleteEvolutionLog godoc
// @Summary Complete evolution log
// @Description Mark an evolution log as completed, failed, or skipped (callback from agent gateway)
// @Tags evolution
// @Param bot_id path string true "Bot ID"
// @Param id path string true "Evolution log ID"
// @Param payload body heartbeat.CompleteEvolutionLogRequest true "Completion payload"
// @Success 200 {object} heartbeat.EvolutionLog
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/evolution-logs/{id}/complete [post]
func (h *HeartbeatHandler) CompleteEvolutionLog(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	logID := c.Param("id")
	if logID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	var req heartbeat.CompleteEvolutionLogRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	item, err := h.engine.CompleteEvolutionLog(c.Request().Context(), logID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, item)
}

// RollbackEvolutionLog restores persona files from the snapshot captured before
// the evolution run identified by :id.
//
// @Summary Rollback evolution log
// @Description Restore bot persona files (IDENTITY.md, SOUL.md, …) to the state captured before the given evolution run
// @Tags evolution
// @Param bot_id path string true "Bot ID"
// @Param id path string true "Evolution log ID"
// @Success 200 {object} rollbackEvolutionResult
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/evolution-logs/{id}/rollback [post]
func (h *HeartbeatHandler) RollbackEvolutionLog(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	logID := strings.TrimSpace(c.Param("id"))
	if logID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	restored, err := h.engine.RollbackEvolution(c.Request().Context(), botID, logID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no snapshot") {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rollbackEvolutionResult{LogID: logID, FilesRestored: restored})
}
