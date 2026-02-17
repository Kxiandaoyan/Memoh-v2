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
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
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
	Embedding ovConfEmbed `json:"embedding"`
	VLM       ovConfVLM   `json:"vlm"`
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

// ensureOVConf generates ov.conf in the bot's data directory, populated from
// bot-specific model settings (falling back to system-wide models).
// The file is always regenerated to stay in sync with the current settings.
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

	conf := h.buildOVConf(ctx, botID)

	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		h.logger.Warn("ov.conf: marshal failed", slog.Any("error", err))
		return
	}
	data = append(data, '\n')

	if err := os.WriteFile(confPath, data, 0o600); err != nil {
		h.logger.Warn("ov.conf: failed to write", slog.Any("error", err))
		return
	}
	h.logger.Info("ov.conf synced from bot settings", slog.String("bot_id", botID), slog.String("path", confPath))
}

// buildOVConf constructs ov.conf content, preferring the bot's own model
// settings and falling back to system-wide models.
func (h *PromptsHandler) buildOVConf(ctx context.Context, botID string) ovConfJSON {
	conf := ovConfJSON{
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

	// Read bot-specific model settings.
	var botEmbeddingModelID, botVlmModelID, botChatModelID string
	if botUUID, err := db.ParseUUID(botID); err == nil {
		if row, err := h.queries.GetSettingsByBotID(ctx, botUUID); err == nil {
			botEmbeddingModelID = row.EmbeddingModelID.String
			botVlmModelID = row.VlmModelID.String
			botChatModelID = row.ChatModelID.String
		}
	}

	// Populate embedding: prefer bot setting, fall back to system first embedding model.
	embPopulated := false
	if botEmbeddingModelID != "" {
		if emb, err := h.modelsService.GetByID(ctx, botEmbeddingModelID); err == nil {
			if provider, provErr := models.FetchProviderByID(ctx, h.queries, emb.LlmProviderID); provErr == nil {
				h.applyEmbeddingConf(&conf, emb, provider)
				embPopulated = true
			}
		}
	}
	if !embPopulated {
		if embModels, err := h.modelsService.ListByType(ctx, models.ModelTypeEmbedding); err == nil && len(embModels) > 0 {
			if provider, provErr := models.FetchProviderByID(ctx, h.queries, embModels[0].LlmProviderID); provErr == nil {
				h.applyEmbeddingConf(&conf, embModels[0], provider)
			}
		}
	}

	// Populate VLM: prefer dedicated vlm_model_id, then fall back to bot chat model,
	// then to system first multimodal chat model.
	vlmPopulated := false
	if botVlmModelID != "" {
		if vlm, err := h.modelsService.GetByID(ctx, botVlmModelID); err == nil {
			if provider, provErr := models.FetchProviderByID(ctx, h.queries, vlm.LlmProviderID); provErr == nil {
				h.applyVLMConf(&conf, vlm, provider)
				vlmPopulated = true
			}
		}
	}
	if !vlmPopulated && botChatModelID != "" {
		if chat, err := h.modelsService.GetByID(ctx, botChatModelID); err == nil {
			if provider, provErr := models.FetchProviderByID(ctx, h.queries, chat.LlmProviderID); provErr == nil {
				h.applyVLMConf(&conf, chat, provider)
				vlmPopulated = true
			}
		}
	}
	if !vlmPopulated {
		if chatModels, err := h.modelsService.ListByType(ctx, models.ModelTypeChat); err == nil && len(chatModels) > 0 {
			selected := &chatModels[0]
			for i := range chatModels {
				if chatModels[i].IsMultimodal {
					selected = &chatModels[i]
					break
				}
			}
			if provider, provErr := models.FetchProviderByID(ctx, h.queries, selected.LlmProviderID); provErr == nil {
				h.applyVLMConf(&conf, *selected, provider)
			}
		}
	}

	return conf
}

func (h *PromptsHandler) applyEmbeddingConf(conf *ovConfJSON, emb models.GetResponse, provider dbsqlc.LlmProvider) {
	conf.Embedding.Dense.APIBase = provider.BaseUrl
	conf.Embedding.Dense.APIKey = provider.ApiKey
	conf.Embedding.Dense.Provider = mapClientTypeToOVProvider(provider.ClientType)
	conf.Embedding.Dense.Model = emb.ModelID
	if emb.Dimensions > 0 {
		conf.Embedding.Dense.Dimension = emb.Dimensions
	}
	h.logger.Info("ov.conf: embedding model resolved",
		slog.String("model", emb.ModelID),
		slog.String("provider", provider.Name))
}

func (h *PromptsHandler) applyVLMConf(conf *ovConfJSON, chat models.GetResponse, provider dbsqlc.LlmProvider) {
	conf.VLM.APIBase = provider.BaseUrl
	conf.VLM.APIKey = provider.ApiKey
	conf.VLM.Provider = mapClientTypeToOVProvider(provider.ClientType)
	conf.VLM.Model = chat.ModelID
	h.logger.Info("ov.conf: VLM model resolved",
		slog.String("model", chat.ModelID),
		slog.String("provider", provider.Name))
}

func (h *PromptsHandler) requireChannelIdentityID(c echo.Context) (string, error) {
	return RequireChannelIdentityID(c)
}

func (h *PromptsHandler) authorizeBotAccess(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error) {
	return AuthorizeBotAccess(ctx, h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: false})
}
