package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
	dbsqlc "github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	"github.com/Kxiandaoyan/Memoh-v2/internal/models"
)

// PromptsHandler manages bot persona/prompt configuration via REST API.
type PromptsHandler struct {
	botService     *bots.Service
	accountService *accounts.Service
	modelsService  *models.Service
	queries        *dbsqlc.Queries
	mcpCfg         config.MCPConfig
	logger         *slog.Logger
}

// NewPromptsHandler creates a PromptsHandler.
func NewPromptsHandler(log *slog.Logger, botService *bots.Service, accountService *accounts.Service, modelsService *models.Service, queries *dbsqlc.Queries, cfg config.Config) *PromptsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &PromptsHandler{
		botService:     botService,
		accountService: accountService,
		modelsService:  modelsService,
		queries:        queries,
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

	// Auto-create ov.conf when OpenViking is enabled, populated from system models.
	if prompts.EnableOpenviking {
		h.ensureOVConf(c.Request().Context(), botID)
	}

	return c.JSON(http.StatusOK, prompts)
}

// ovConfJSON is the structure for ov.conf.
type ovConfJSON struct {
	README    []string       `json:"_README"`
	Embedding ovConfEmbed    `json:"embedding"`
	VLM       ovConfVLM      `json:"vlm"`
}

type ovConfEmbed struct {
	Dense ovConfDense `json:"dense"`
}

type ovConfDense struct {
	APIBase   string `json:"api_base"`
	APIKey    string `json:"api_key"`
	Provider  string `json:"provider"`
	Dimension int    `json:"dimension"`
	Model     string `json:"model"`
}

type ovConfVLM struct {
	APIBase    string `json:"api_base"`
	APIKey     string `json:"api_key"`
	Provider   string `json:"provider"`
	MaxRetries int    `json:"max_retries"`
	Model      string `json:"model"`
}

// mapClientTypeToOVProvider maps Memoh's client_type to OpenViking's provider string.
func mapClientTypeToOVProvider(clientType string) string {
	switch strings.ToLower(clientType) {
	case "openai", "azure", "ollama":
		return "openai"
	case "volcengine", "dashscope":
		return "volcengine"
	default:
		return "openai"
	}
}

// ensureOVConf creates ov.conf in the bot's data directory, auto-populated from
// system-configured embedding and chat models. If the file already exists, it is
// not overwritten (user may have customized it).
func (h *PromptsHandler) ensureOVConf(ctx context.Context, botID string) {
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

	conf := h.buildOVConf(ctx)

	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		h.logger.Warn("ov.conf: marshal failed", slog.Any("error", err))
		return
	}
	data = append(data, '\n')

	if err := os.WriteFile(confPath, data, 0o644); err != nil {
		h.logger.Warn("ov.conf: failed to create", slog.Any("error", err))
		return
	}
	h.logger.Info("ov.conf created from system models", slog.String("bot_id", botID), slog.String("path", confPath))
}

// buildOVConf constructs ov.conf content from system-configured models/providers.
func (h *PromptsHandler) buildOVConf(ctx context.Context) ovConfJSON {
	conf := ovConfJSON{
		README: []string{
			"=== OpenViking Configuration ===",
			"Auto-generated from system Models & Providers settings.",
			"Edit via web UI: Bot Detail → Files → ov.conf",
			"",
			"embedding.dense — Vector embedding model (semantic search)",
			"vlm — Vision/Language model (content understanding)",
			"provider: 'openai' (also for compatible APIs) or 'volcengine'",
		},
		Embedding: ovConfEmbed{
			Dense: ovConfDense{
				APIBase:   "https://api.openai.com/v1",
				APIKey:    "sk-your-api-key-here",
				Provider:  "openai",
				Dimension: 1536,
				Model:     "text-embedding-3-small",
			},
		},
		VLM: ovConfVLM{
			APIBase:    "https://api.openai.com/v1",
			APIKey:     "sk-your-api-key-here",
			Provider:   "openai",
			MaxRetries: 2,
			Model:      "gpt-4o",
		},
	}

	if h.modelsService == nil || h.queries == nil {
		return conf
	}

	// Try to populate embedding from system's first embedding model.
	embModels, err := h.modelsService.ListByType(ctx, models.ModelTypeEmbedding)
	if err == nil && len(embModels) > 0 {
		emb := embModels[0]
		provider, provErr := models.FetchProviderByID(ctx, h.queries, emb.LlmProviderID)
		if provErr == nil {
			conf.Embedding.Dense.APIBase = provider.BaseUrl
			conf.Embedding.Dense.APIKey = provider.ApiKey
			conf.Embedding.Dense.Provider = mapClientTypeToOVProvider(provider.ClientType)
			conf.Embedding.Dense.Model = emb.ModelID
			if emb.Dimensions > 0 {
				conf.Embedding.Dense.Dimension = emb.Dimensions
			}
			h.logger.Info("ov.conf: embedding from system",
				slog.String("model", emb.ModelID),
				slog.String("provider", provider.Name))
		}
	}

	// Try to populate VLM from system's first multimodal chat model (prefer multimodal).
	chatModels, err := h.modelsService.ListByType(ctx, models.ModelTypeChat)
	if err == nil && len(chatModels) > 0 {
		var selected *models.GetResponse
		for i := range chatModels {
			if chatModels[i].IsMultimodal {
				selected = &chatModels[i]
				break
			}
		}
		if selected == nil {
			selected = &chatModels[0]
		}
		provider, provErr := models.FetchProviderByID(ctx, h.queries, selected.LlmProviderID)
		if provErr == nil {
			conf.VLM.APIBase = provider.BaseUrl
			conf.VLM.APIKey = provider.ApiKey
			conf.VLM.Provider = mapClientTypeToOVProvider(provider.ClientType)
			conf.VLM.Model = selected.ModelID
			h.logger.Info("ov.conf: VLM from system",
				slog.String("model", selected.ModelID),
				slog.String("provider", provider.Name))
		}
	}

	return conf
}

func (h *PromptsHandler) requireChannelIdentityID(c echo.Context) (string, error) {
	return RequireChannelIdentityID(c)
}

func (h *PromptsHandler) authorizeBotAccess(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error) {
	return AuthorizeBotAccess(ctx, h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: false})
}
