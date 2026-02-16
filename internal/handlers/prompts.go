package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
)

// PromptsHandler manages bot persona/prompt configuration via REST API.
type PromptsHandler struct {
	botService     *bots.Service
	accountService *accounts.Service
	logger         *slog.Logger
}

// NewPromptsHandler creates a PromptsHandler.
func NewPromptsHandler(log *slog.Logger, botService *bots.Service, accountService *accounts.Service) *PromptsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &PromptsHandler{
		botService:     botService,
		accountService: accountService,
		logger:         log.With(slog.String("handler", "prompts")),
	}
}

// Register registers prompt routes.
func (h *PromptsHandler) Register(e *echo.Echo) {
	group := e.Group("/bots/:bot_id/prompts")
	group.GET("", h.Get)
	group.PUT("", h.Update)
}

// Get godoc
// @Summary Get bot prompts
// @Description Get persona/prompt configuration for a bot
// @Tags prompts
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} bots.Prompts
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/prompts [get]
func (h *PromptsHandler) Get(c echo.Context) error {
	channelIdentityID, err := h.requireChannelIdentityID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), channelIdentityID, botID); err != nil {
		return err
	}
	prompts, err := h.botService.GetPrompts(c.Request().Context(), botID)
	if err != nil {
		h.logger.Error("failed to get bot prompts", slog.String("bot_id", botID), slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get bot prompts")
	}
	return c.JSON(http.StatusOK, prompts)
}

// Update godoc
// @Summary Update bot prompts
// @Description Update persona/prompt configuration for a bot
// @Tags prompts
// @Param bot_id path string true "Bot ID"
// @Param payload body bots.UpdatePromptsRequest true "Prompts payload"
// @Success 200 {object} bots.Prompts
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/prompts [put]
func (h *PromptsHandler) Update(c echo.Context) error {
	channelIdentityID, err := h.requireChannelIdentityID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), channelIdentityID, botID); err != nil {
		return err
	}
	var req bots.UpdatePromptsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	prompts, err := h.botService.UpdatePrompts(c.Request().Context(), botID, req)
	if err != nil {
		h.logger.Error("failed to update bot prompts", slog.String("bot_id", botID), slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update bot prompts")
	}
	return c.JSON(http.StatusOK, prompts)
}

func (h *PromptsHandler) requireChannelIdentityID(c echo.Context) (string, error) {
	return RequireChannelIdentityID(c)
}

func (h *PromptsHandler) authorizeBotAccess(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error) {
	return AuthorizeBotAccess(ctx, h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: false})
}
