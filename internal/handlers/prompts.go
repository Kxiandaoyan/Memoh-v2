package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
)

// PromptsHandler manages bot persona/prompt configuration via REST API.
type PromptsHandler struct {
	botService     *bots.Service
	accountService *accounts.Service
	mcpCfg         config.MCPConfig
	logger         *slog.Logger
}

// NewPromptsHandler creates a PromptsHandler.
func NewPromptsHandler(log *slog.Logger, botService *bots.Service, accountService *accounts.Service, cfg config.Config) *PromptsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &PromptsHandler{
		botService:     botService,
		accountService: accountService,
		mcpCfg:         cfg.MCP,
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

	// Auto-create ov.conf template when OpenViking is enabled.
	if prompts.EnableOpenviking {
		h.ensureOVConf(botID)
	}

	return c.JSON(http.StatusOK, prompts)
}

// ensureOVConf creates a default ov.conf in the bot's data directory if it
// does not exist yet. This gives the user a template they can fill in via the
// Files UI without having to craft the JSON structure from scratch.
func (h *PromptsHandler) ensureOVConf(botID string) {
	dataRoot := strings.TrimSpace(h.mcpCfg.DataRoot)
	if dataRoot == "" {
		dataRoot = config.DefaultDataRoot
	}
	absRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		h.logger.Warn("ov.conf: cannot resolve data root", slog.Any("error", err))
		return
	}
	botDir := filepath.Join(absRoot, "bots", botID)
	if err := os.MkdirAll(botDir, 0o755); err != nil {
		h.logger.Warn("ov.conf: cannot create bot dir", slog.Any("error", err))
		return
	}
	confPath := filepath.Join(botDir, "ov.conf")
	if _, err := os.Stat(confPath); err == nil {
		return // already exists, don't overwrite
	}

	const defaultOVConf = `{
  "_README": [
    "=== OpenViking Configuration ===",
    "Edit this file via web UI: Bot Detail → Files → ov.conf",
    "",
    "embedding.dense — Vector embedding model (for semantic search)",
    "  api_base   : LLM provider API endpoint (e.g. https://api.openai.com/v1)",
    "  api_key    : Your API key (e.g. sk-xxxx)",
    "  provider   : 'openai' or 'volcengine'",
    "  dimension  : Vector dimensions — 1536 for text-embedding-3-small, 1024 for doubao-embedding",
    "  model      : Model name — e.g. text-embedding-3-small, doubao-embedding-vision-250615",
    "  input      : (optional) Set to 'multimodal' when using doubao-embedding-vision",
    "",
    "vlm — Vision Language Model (for content understanding)",
    "  api_base   : Same or different API endpoint",
    "  api_key    : Same or different API key",
    "  provider   : 'openai' or 'volcengine'",
    "  model      : VLM model — e.g. gpt-4o, doubao-seed-1-8-251228",
    "  max_retries: Retry count on failure (default 2)"
  ],
  "embedding": {
    "dense": {
      "api_base": "https://api.openai.com/v1",
      "api_key": "sk-your-api-key-here",
      "provider": "openai",
      "dimension": 1536,
      "model": "text-embedding-3-small"
    }
  },
  "vlm": {
    "api_base": "https://api.openai.com/v1",
    "api_key": "sk-your-api-key-here",
    "provider": "openai",
    "max_retries": 2,
    "model": "gpt-4o"
  }
}
`
	if err := os.WriteFile(confPath, []byte(defaultOVConf), 0o644); err != nil {
		h.logger.Warn("ov.conf: failed to create template", slog.Any("error", err))
		return
	}
	h.logger.Info("ov.conf template created", slog.String("bot_id", botID), slog.String("path", confPath))
}

func (h *PromptsHandler) requireChannelIdentityID(c echo.Context) (string, error) {
	return RequireChannelIdentityID(c)
}

func (h *PromptsHandler) authorizeBotAccess(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error) {
	return AuthorizeBotAccess(ctx, h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: false})
}
