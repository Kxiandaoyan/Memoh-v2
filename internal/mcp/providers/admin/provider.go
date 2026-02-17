package admin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"

	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/models"
	"github.com/Kxiandaoyan/Memoh-v2/internal/providers"
)

const (
	toolAdminListBots      = "admin_list_bots"
	toolAdminCreateBot     = "admin_create_bot"
	toolAdminDeleteBot     = "admin_delete_bot"
	toolAdminListModels    = "admin_list_models"
	toolAdminCreateModel   = "admin_create_model"
	toolAdminDeleteModel   = "admin_delete_model"
	toolAdminListProviders = "admin_list_providers"
	toolAdminCreateProvider = "admin_create_provider"
	toolAdminUpdateProvider = "admin_update_provider"
)

// BotService is the subset of bots.Service used by the admin provider.
type BotService interface {
	ListByOwner(ctx context.Context, ownerUserID string) ([]bots.Bot, error)
	Create(ctx context.Context, ownerUserID string, req bots.CreateBotRequest) (bots.Bot, error)
	Delete(ctx context.Context, botID string) error
	Get(ctx context.Context, botID string) (bots.Bot, error)
}

// ModelService is the subset of models.Service used by the admin provider.
type ModelService interface {
	List(ctx context.Context) ([]models.GetResponse, error)
	Create(ctx context.Context, req models.AddRequest) (models.AddResponse, error)
	DeleteByID(ctx context.Context, id string) error
}

// ProviderService is the subset of providers.Service used by the admin provider.
type ProviderService interface {
	List(ctx context.Context) ([]providers.GetResponse, error)
	Create(ctx context.Context, req providers.CreateRequest) (providers.GetResponse, error)
	Update(ctx context.Context, id string, req providers.UpdateRequest) (providers.GetResponse, error)
}

// OwnerResolver resolves the owner user ID for a given bot.
type OwnerResolver interface {
	Get(ctx context.Context, botID string) (bots.Bot, error)
}

// Executor implements mcpgw.ToolExecutor for admin management tools.
// It is only registered for privileged bots.
type Executor struct {
	botService      BotService
	modelService    ModelService
	providerService ProviderService
	ownerResolver   OwnerResolver
	logger          *slog.Logger
}

// NewExecutor creates an admin tool executor.
func NewExecutor(
	log *slog.Logger,
	botSvc BotService,
	modelSvc ModelService,
	providerSvc ProviderService,
) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		botService:      botSvc,
		modelService:    modelSvc,
		providerService: providerSvc,
		ownerResolver:   botSvc,
		logger:          log.With(slog.String("provider", "admin_tool")),
	}
}

func (p *Executor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolAdminListBots,
			Description: "List all bots owned by your owner. Use this to see existing bots.",
			InputSchema: emptyObjectSchema(),
		},
		{
			Name:        toolAdminCreateBot,
			Description: "Create a new bot. The new bot will be owned by your owner.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"display_name": map[string]any{"type": "string", "description": "Bot display name"},
					"type":         map[string]any{"type": "string", "enum": []string{"personal", "public"}, "description": "Bot type: personal (owner-only) or public (multi-user)"},
				},
				"required": []string{"display_name", "type"},
			},
		},
		{
			Name:        toolAdminDeleteBot,
			Description: "Delete a bot by ID. This will remove the bot and all its data.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"bot_id": map[string]any{"type": "string", "description": "Bot ID to delete"},
				},
				"required": []string{"bot_id"},
			},
		},
		{
			Name:        toolAdminListModels,
			Description: "List all configured AI models (chat, memory, embedding).",
			InputSchema: emptyObjectSchema(),
		},
		{
			Name:        toolAdminCreateModel,
			Description: "Create a new AI model configuration.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"model_id":        map[string]any{"type": "string", "description": "Model identifier (e.g. gpt-4o, claude-3.5-sonnet)"},
					"name":            map[string]any{"type": "string", "description": "Display name"},
					"llm_provider_id": map[string]any{"type": "string", "description": "UUID of the LLM provider"},
					"type":            map[string]any{"type": "string", "enum": []string{"chat", "embedding"}, "description": "Model type"},
					"is_multimodal":   map[string]any{"type": "boolean", "description": "Whether the model supports images"},
					"dimensions":      map[string]any{"type": "integer", "description": "Embedding dimensions (required for embedding type)"},
					"context_window":  map[string]any{"type": "integer", "description": "Context window size in tokens"},
				},
				"required": []string{"model_id", "llm_provider_id", "type"},
			},
		},
		{
			Name:        toolAdminDeleteModel,
			Description: "Delete a model by its internal UUID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "Model internal UUID"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        toolAdminListProviders,
			Description: "List all LLM providers (e.g. OpenAI, Anthropic, Ollama).",
			InputSchema: emptyObjectSchema(),
		},
		{
			Name:        toolAdminCreateProvider,
			Description: "Create a new LLM provider configuration.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string", "description": "Provider name (e.g. openai, anthropic)"},
					"client_type": map[string]any{"type": "string", "description": "Client type: openai or anthropic"},
					"base_url":    map[string]any{"type": "string", "description": "API base URL"},
					"api_key":     map[string]any{"type": "string", "description": "API key"},
				},
				"required": []string{"name", "client_type", "base_url"},
			},
		},
		{
			Name:        toolAdminUpdateProvider,
			Description: "Update an existing LLM provider configuration.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":          map[string]any{"type": "string", "description": "Provider UUID"},
					"name":        map[string]any{"type": "string"},
					"client_type": map[string]any{"type": "string"},
					"base_url":    map[string]any{"type": "string"},
					"api_key":     map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
		},
	}, nil
}

func (p *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}

	ownerUserID, err := p.resolveOwner(ctx, botID)
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("resolve owner: %v", err)), nil
	}

	switch toolName {
	case toolAdminListBots:
		return p.listBots(ctx, ownerUserID)
	case toolAdminCreateBot:
		return p.createBot(ctx, ownerUserID, arguments)
	case toolAdminDeleteBot:
		return p.deleteBot(ctx, arguments)
	case toolAdminListModels:
		return p.listModels(ctx)
	case toolAdminCreateModel:
		return p.createModel(ctx, arguments)
	case toolAdminDeleteModel:
		return p.deleteModel(ctx, arguments)
	case toolAdminListProviders:
		return p.listProviders(ctx)
	case toolAdminCreateProvider:
		return p.createProvider(ctx, arguments)
	case toolAdminUpdateProvider:
		return p.updateProvider(ctx, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

func (p *Executor) resolveOwner(ctx context.Context, botID string) (string, error) {
	bot, err := p.ownerResolver.Get(ctx, botID)
	if err != nil {
		return "", err
	}
	return bot.OwnerUserID, nil
}

func (p *Executor) listBots(ctx context.Context, ownerUserID string) (map[string]any, error) {
	items, err := p.botService.ListByOwner(ctx, ownerUserID)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, b := range items {
		result = append(result, map[string]any{
			"id":           b.ID,
			"display_name": b.DisplayName,
			"type":         b.Type,
			"status":       b.Status,
			"is_active":    b.IsActive,
		})
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{"bots": result}), nil
}

func (p *Executor) createBot(ctx context.Context, ownerUserID string, args map[string]any) (map[string]any, error) {
	displayName := mcpgw.StringArg(args, "display_name")
	botType := mcpgw.StringArg(args, "type")
	if displayName == "" || botType == "" {
		return mcpgw.BuildToolErrorResult("display_name and type are required"), nil
	}
	bot, err := p.botService.Create(ctx, ownerUserID, bots.CreateBotRequest{
		Type:        botType,
		DisplayName: displayName,
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{
		"id":           bot.ID,
		"display_name": bot.DisplayName,
		"type":         bot.Type,
		"status":       bot.Status,
	}), nil
}

func (p *Executor) deleteBot(ctx context.Context, args map[string]any) (map[string]any, error) {
	botID := mcpgw.StringArg(args, "bot_id")
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}
	if err := p.botService.Delete(ctx, botID); err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{"success": true, "bot_id": botID}), nil
}

func (p *Executor) listModels(ctx context.Context) (map[string]any, error) {
	items, err := p.modelService.List(ctx)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, m := range items {
		result = append(result, map[string]any{
			"model_id":        m.ModelID,
			"name":            m.Name,
			"type":            m.Type,
			"llm_provider_id": m.LlmProviderID,
			"is_multimodal":   m.IsMultimodal,
			"dimensions":      m.Dimensions,
			"context_window":  m.ContextWindow,
		})
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{"models": result}), nil
}

func (p *Executor) createModel(ctx context.Context, args map[string]any) (map[string]any, error) {
	modelID := mcpgw.StringArg(args, "model_id")
	providerID := mcpgw.StringArg(args, "llm_provider_id")
	modelType := mcpgw.StringArg(args, "type")
	if modelID == "" || providerID == "" || modelType == "" {
		return mcpgw.BuildToolErrorResult("model_id, llm_provider_id, and type are required"), nil
	}
	req := models.AddRequest{
		ModelID:       modelID,
		Name:          mcpgw.StringArg(args, "name"),
		LlmProviderID: providerID,
		Type:          models.ModelType(modelType),
	}
	if multimodal, ok, _ := mcpgw.BoolArg(args, "is_multimodal"); ok {
		req.IsMultimodal = multimodal
	}
	if dims, ok, _ := mcpgw.IntArg(args, "dimensions"); ok {
		req.Dimensions = dims
	}
	if ctxWin, ok, _ := mcpgw.IntArg(args, "context_window"); ok {
		req.ContextWindow = ctxWin
	}
	resp, err := p.modelService.Create(ctx, req)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{
		"id":       resp.ID,
		"model_id": resp.ModelID,
	}), nil
}

func (p *Executor) deleteModel(ctx context.Context, args map[string]any) (map[string]any, error) {
	id := mcpgw.StringArg(args, "id")
	if id == "" {
		return mcpgw.BuildToolErrorResult("id is required"), nil
	}
	if err := p.modelService.DeleteByID(ctx, id); err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{"success": true}), nil
}

func (p *Executor) listProviders(ctx context.Context) (map[string]any, error) {
	items, err := p.providerService.List(ctx)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, prov := range items {
		result = append(result, map[string]any{
			"id":          prov.ID,
			"name":        prov.Name,
			"client_type": prov.ClientType,
			"base_url":    prov.BaseURL,
		})
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{"providers": result}), nil
}

func (p *Executor) createProvider(ctx context.Context, args map[string]any) (map[string]any, error) {
	name := mcpgw.StringArg(args, "name")
	clientType := mcpgw.StringArg(args, "client_type")
	baseURL := mcpgw.StringArg(args, "base_url")
	if name == "" || clientType == "" || baseURL == "" {
		return mcpgw.BuildToolErrorResult("name, client_type, and base_url are required"), nil
	}
	resp, err := p.providerService.Create(ctx, providers.CreateRequest{
		Name:       name,
		ClientType: providers.ClientType(clientType),
		BaseURL:    baseURL,
		APIKey:     mcpgw.StringArg(args, "api_key"),
	})
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{
		"id":   resp.ID,
		"name": resp.Name,
	}), nil
}

func (p *Executor) updateProvider(ctx context.Context, args map[string]any) (map[string]any, error) {
	id := mcpgw.StringArg(args, "id")
	if id == "" {
		return mcpgw.BuildToolErrorResult("id is required"), nil
	}
	req := providers.UpdateRequest{}
	if v := mcpgw.StringArg(args, "name"); v != "" {
		req.Name = &v
	}
	if v := mcpgw.StringArg(args, "client_type"); v != "" {
		ct := providers.ClientType(v)
		req.ClientType = &ct
	}
	if v := mcpgw.StringArg(args, "base_url"); v != "" {
		req.BaseURL = &v
	}
	if v := mcpgw.StringArg(args, "api_key"); v != "" {
		req.APIKey = &v
	}
	resp, err := p.providerService.Update(ctx, id, req)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{
		"id":   resp.ID,
		"name": resp.Name,
	}), nil
}

func emptyObjectSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
