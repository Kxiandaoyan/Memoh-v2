package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/heartbeat"
)

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

func (h *HeartbeatHandler) requireUserID(c echo.Context) (string, error) {
	return RequireChannelIdentityID(c)
}

func (h *HeartbeatHandler) authorizeBotAccess(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error) {
	return AuthorizeBotAccess(ctx, h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: false})
}
