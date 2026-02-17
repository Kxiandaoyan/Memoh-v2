package models

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
)

// Service provides CRUD operations for models
type Service struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

// NewService creates a new models service
func NewService(log *slog.Logger, queries *sqlc.Queries) *Service {
	return &Service{
		queries: queries,
		logger:  log.With(slog.String("service", "models")),
	}
}

// Create adds a new model to the database
func (s *Service) Create(ctx context.Context, req AddRequest) (AddResponse, error) {
	model := Model(req)
	if err := model.Validate(); err != nil {
		return AddResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	// Convert to sqlc params
	llmProviderID, err := db.ParseUUID(model.LlmProviderID)
	if err != nil {
		return AddResponse{}, fmt.Errorf("invalid llm provider ID: %w", err)
	}

	contextWindow := int32(model.ContextWindow)
	if contextWindow <= 0 {
		contextWindow = 128000
	}

	params := sqlc.CreateModelParams{
		ModelID:       model.ModelID,
		LlmProviderID: llmProviderID,
		IsMultimodal:  model.IsMultimodal,
		Type:          string(model.Type),
		ContextWindow: contextWindow,
		Reasoning:     model.Reasoning,
		MaxTokens:     int32(model.MaxTokens),
	}

	// Handle optional name field
	if model.Name != "" {
		params.Name = pgtype.Text{String: model.Name, Valid: true}
	}

	// Handle optional dimensions field (only for embedding models)
	if model.Type == ModelTypeEmbedding && model.Dimensions > 0 {
		params.Dimensions = pgtype.Int4{Int32: int32(model.Dimensions), Valid: true}
	}

	if model.FallbackModelID != "" {
		fbUUID, err := s.resolveFallbackModelID(ctx, model.FallbackModelID, model.ModelID)
		if err != nil {
			return AddResponse{}, err
		}
		params.FallbackModelID = fbUUID
	}

	created, err := s.queries.CreateModel(ctx, params)
	if err != nil {
		return AddResponse{}, fmt.Errorf("failed to create model: %w", err)
	}

	// Convert pgtype.UUID to string
	var idStr string
	if created.ID.Valid {
		id, err := uuid.FromBytes(created.ID.Bytes[:])
		if err != nil {
			return AddResponse{}, fmt.Errorf("failed to convert UUID: %w", err)
		}
		idStr = id.String()
	}

	return AddResponse{
		ID:      idStr,
		ModelID: created.ModelID,
	}, nil
}

// GetByID retrieves a model by its internal UUID
func (s *Service) GetByID(ctx context.Context, id string) (GetResponse, error) {
	uuid, err := db.ParseUUID(id)
	if err != nil {
		return GetResponse{}, fmt.Errorf("invalid ID: %w", err)
	}

	dbModel, err := s.queries.GetModelByID(ctx, uuid)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to get model: %w", err)
	}

	return s.convertToGetResponse(ctx, dbModel), nil
}

// GetByModelID retrieves a model by its model_id field
func (s *Service) GetByModelID(ctx context.Context, modelID string) (GetResponse, error) {
	if modelID == "" {
		return GetResponse{}, fmt.Errorf("model_id is required")
	}

	dbModel, err := s.queries.GetModelByModelID(ctx, modelID)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to get model: %w", err)
	}

	return s.convertToGetResponse(ctx, dbModel), nil
}

// List returns all models
func (s *Service) List(ctx context.Context) ([]GetResponse, error) {
	dbModels, err := s.queries.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	return s.convertToGetResponseList(ctx, dbModels), nil
}

// ListByType returns models filtered by type (chat or embedding)
func (s *Service) ListByType(ctx context.Context, modelType ModelType) ([]GetResponse, error) {
	if modelType != ModelTypeChat && modelType != ModelTypeEmbedding {
		return nil, fmt.Errorf("invalid model type: %s", modelType)
	}

	dbModels, err := s.queries.ListModelsByType(ctx, string(modelType))
	if err != nil {
		return nil, fmt.Errorf("failed to list models by type: %w", err)
	}

	return s.convertToGetResponseList(ctx, dbModels), nil
}

// ListByClientType returns models filtered by client type
func (s *Service) ListByClientType(ctx context.Context, clientType ClientType) ([]GetResponse, error) {
	if !isValidClientType(clientType) {
		return nil, fmt.Errorf("invalid client type: %s", clientType)
	}

	dbModels, err := s.queries.ListModelsByClientType(ctx, string(clientType))
	if err != nil {
		return nil, fmt.Errorf("failed to list models by client type: %w", err)
	}

	return s.convertToGetResponseList(ctx, dbModels), nil
}

// ListByProviderID returns models filtered by provider ID.
func (s *Service) ListByProviderID(ctx context.Context, providerID string) ([]GetResponse, error) {
	if strings.TrimSpace(providerID) == "" {
		return nil, fmt.Errorf("provider id is required")
	}
	uuid, err := db.ParseUUID(providerID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider id: %w", err)
	}
	dbModels, err := s.queries.ListModelsByProviderID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to list models by provider: %w", err)
	}
	return s.convertToGetResponseList(ctx, dbModels), nil
}

// ListByProviderIDAndType returns models filtered by provider ID and type.
func (s *Service) ListByProviderIDAndType(ctx context.Context, providerID string, modelType ModelType) ([]GetResponse, error) {
	if modelType != ModelTypeChat && modelType != ModelTypeEmbedding {
		return nil, fmt.Errorf("invalid model type: %s", modelType)
	}
	if strings.TrimSpace(providerID) == "" {
		return nil, fmt.Errorf("provider id is required")
	}
	uuid, err := db.ParseUUID(providerID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider id: %w", err)
	}
	dbModels, err := s.queries.ListModelsByProviderIDAndType(ctx, sqlc.ListModelsByProviderIDAndTypeParams{
		LlmProviderID: uuid,
		Type:          string(modelType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list models by provider and type: %w", err)
	}
	return s.convertToGetResponseList(ctx, dbModels), nil
}

// UpdateByID updates a model by its internal UUID
func (s *Service) UpdateByID(ctx context.Context, id string, req UpdateRequest) (GetResponse, error) {
	uuid, err := db.ParseUUID(id)
	if err != nil {
		return GetResponse{}, fmt.Errorf("invalid ID: %w", err)
	}

	model := Model(req)
	if err := model.Validate(); err != nil {
		return GetResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	contextWindow := int32(model.ContextWindow)
	if contextWindow <= 0 {
		contextWindow = 128000
	}

	params := sqlc.UpdateModelParams{
		ID:            uuid,
		IsMultimodal:  model.IsMultimodal,
		Type:          string(model.Type),
		ContextWindow: contextWindow,
		Reasoning:     model.Reasoning,
		MaxTokens:     int32(model.MaxTokens),
	}

	llmProviderID, err := db.ParseUUID(model.LlmProviderID)
	if err != nil {
		return GetResponse{}, fmt.Errorf("invalid llm provider ID: %w", err)
	}
	params.LlmProviderID = llmProviderID

	if model.Name != "" {
		params.Name = pgtype.Text{String: model.Name, Valid: true}
	}

	if model.Type == ModelTypeEmbedding && model.Dimensions > 0 {
		params.Dimensions = pgtype.Int4{Int32: int32(model.Dimensions), Valid: true}
	}

	if model.FallbackModelID != "" {
		fbUUID, err := s.resolveFallbackModelID(ctx, model.FallbackModelID, model.ModelID)
		if err != nil {
			return GetResponse{}, err
		}
		params.FallbackModelID = fbUUID
	}

	updated, err := s.queries.UpdateModel(ctx, params)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to update model: %w", err)
	}

	return s.convertToGetResponse(ctx, updated), nil
}

// UpdateByModelID updates a model by its model_id field
func (s *Service) UpdateByModelID(ctx context.Context, modelID string, req UpdateRequest) (GetResponse, error) {
	if modelID == "" {
		return GetResponse{}, fmt.Errorf("model_id is required")
	}

	model := Model(req)
	if err := model.Validate(); err != nil {
		return GetResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	contextWindow := int32(model.ContextWindow)
	if contextWindow <= 0 {
		contextWindow = 128000
	}

	params := sqlc.UpdateModelByModelIDParams{
		ModelID:       modelID,
		NewModelID:    model.ModelID,
		IsMultimodal:  model.IsMultimodal,
		Type:          string(model.Type),
		ContextWindow: contextWindow,
		Reasoning:     model.Reasoning,
		MaxTokens:     int32(model.MaxTokens),
	}

	llmProviderID, err := db.ParseUUID(model.LlmProviderID)
	if err != nil {
		return GetResponse{}, fmt.Errorf("invalid llm provider ID: %w", err)
	}
	params.LlmProviderID = llmProviderID

	if model.Name != "" {
		params.Name = pgtype.Text{String: model.Name, Valid: true}
	}

	if model.Type == ModelTypeEmbedding && model.Dimensions > 0 {
		params.Dimensions = pgtype.Int4{Int32: int32(model.Dimensions), Valid: true}
	}

	if model.FallbackModelID != "" {
		fbUUID, err := s.resolveFallbackModelID(ctx, model.FallbackModelID, model.ModelID)
		if err != nil {
			return GetResponse{}, err
		}
		params.FallbackModelID = fbUUID
	}

	updated, err := s.queries.UpdateModelByModelID(ctx, params)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to update model: %w", err)
	}

	return s.convertToGetResponse(ctx, updated), nil
}

// DeleteByID deletes a model by its internal UUID
func (s *Service) DeleteByID(ctx context.Context, id string) error {
	uuid, err := db.ParseUUID(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	if err := s.queries.DeleteModel(ctx, uuid); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

// DeleteByModelID deletes a model by its model_id field
func (s *Service) DeleteByModelID(ctx context.Context, modelID string) error {
	if modelID == "" {
		return fmt.Errorf("model_id is required")
	}

	if err := s.queries.DeleteModelByModelID(ctx, modelID); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

// Count returns the total number of models
func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.queries.CountModels(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count models: %w", err)
	}
	return count, nil
}

// CountByType returns the number of models of a specific type
func (s *Service) CountByType(ctx context.Context, modelType ModelType) (int64, error) {
	if modelType != ModelTypeChat && modelType != ModelTypeEmbedding {
		return 0, fmt.Errorf("invalid model type: %s", modelType)
	}

	count, err := s.queries.CountModelsByType(ctx, string(modelType))
	if err != nil {
		return 0, fmt.Errorf("failed to count models by type: %w", err)
	}
	return count, nil
}

// Helper functions

func (s *Service) convertToGetResponse(ctx context.Context, dbModel sqlc.Model) GetResponse {
	resp := GetResponse{
		ModelID: dbModel.ModelID,
		Model: Model{
			ModelID:       dbModel.ModelID,
			IsMultimodal:  dbModel.IsMultimodal,
			Input:         modelInputFromMultimodal(dbModel.IsMultimodal),
			Type:          ModelType(dbModel.Type),
			ContextWindow: int(dbModel.ContextWindow),
			Reasoning:     dbModel.Reasoning,
			MaxTokens:     int(dbModel.MaxTokens),
		},
	}

	if dbModel.LlmProviderID.Valid {
		resp.Model.LlmProviderID = dbModel.LlmProviderID.String()
	}

	if dbModel.Name.Valid {
		resp.Model.Name = dbModel.Name.String
	}

	if dbModel.Dimensions.Valid {
		resp.Model.Dimensions = int(dbModel.Dimensions.Int32)
	}

	if dbModel.FallbackModelID.Valid {
		// Resolve the internal UUID to the human-readable model_id so the
		// frontend dropdown can match it directly.
		if fbModel, err := s.queries.GetModelByID(ctx, dbModel.FallbackModelID); err == nil {
			resp.Model.FallbackModelID = fbModel.ModelID
		} else {
			resp.Model.FallbackModelID = dbModel.FallbackModelID.String()
		}
	}

	return resp
}

// resolveFallbackModelID converts a fallback_model_id value to a pgtype.UUID.
// It accepts either a UUID string (internal ID) or a model_id string (e.g. "gpt-4o-mini").
// selfModelID is the model_id of the model being created/updated — used to
// prevent circular self-references.
func (s *Service) resolveFallbackModelID(ctx context.Context, raw string, selfModelID string) (pgtype.UUID, error) {
	if raw == "" {
		return pgtype.UUID{}, nil
	}

	// Prevent direct self-reference (A → A).
	if raw == selfModelID {
		return pgtype.UUID{}, fmt.Errorf("fallback model cannot reference itself")
	}

	// Try parsing as UUID first.
	if parsed, err := db.ParseUUID(raw); err == nil {
		// Also check: does the resolved UUID's model have a fallback pointing
		// back to selfModelID? (A → B → A)
		if fbModel, lookupErr := s.queries.GetModelByID(ctx, parsed); lookupErr == nil {
			if fbModel.ModelID == selfModelID {
				return pgtype.UUID{}, fmt.Errorf("circular fallback: %s → %s → %s", selfModelID, raw, selfModelID)
			}
		}
		return parsed, nil
	}

	// Treat as model_id — look up the internal UUID.
	dbModel, err := s.queries.GetModelByModelID(ctx, raw)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("fallback model %q not found: %w", raw, err)
	}

	// Check one level: does the fallback model's own fallback point back?
	if dbModel.FallbackModelID.Valid {
		if fbOfFb, lookupErr := s.queries.GetModelByID(ctx, dbModel.FallbackModelID); lookupErr == nil {
			if fbOfFb.ModelID == selfModelID {
				return pgtype.UUID{}, fmt.Errorf("circular fallback: %s → %s → %s", selfModelID, raw, selfModelID)
			}
		}
	}

	return dbModel.ID, nil
}

func (s *Service) convertToGetResponseList(ctx context.Context, dbModels []sqlc.Model) []GetResponse {
	responses := make([]GetResponse, 0, len(dbModels))
	for _, dbModel := range dbModels {
		responses = append(responses, s.convertToGetResponse(ctx, dbModel))
	}
	return responses
}

// modelInputFromMultimodal builds the input list based on multimodal support.
func modelInputFromMultimodal(isMultimodal bool) []string {
	if isMultimodal {
		return []string{ModelInputText, ModelInputImage}
	}
	return []string{ModelInputText}
}

func isValidClientType(clientType ClientType) bool {
	switch clientType {
	case ClientTypeOpenAI,
		ClientTypeOpenAICompat,
		ClientTypeAnthropic,
		ClientTypeGoogle,
		ClientTypeAzure,
		ClientTypeBedrock,
		ClientTypeMistral,
		ClientTypeXAI,
		ClientTypeOllama,
		ClientTypeDashscope,
		ClientTypeDeepSeek,
		ClientTypeZaiGlobal,
		ClientTypeZaiCN,
		ClientTypeZaiCodingGlobal,
		ClientTypeZaiCodingCN,
		ClientTypeMinimaxGlobal,
		ClientTypeMinimaxCN,
		ClientTypeMoonshotGlobal,
		ClientTypeMoonshotCN,
		ClientTypeVolcengine,
		ClientTypeVolcengineCoding,
		ClientTypeQianfan,
		ClientTypeGroq,
		ClientTypeOpenRouter,
		ClientTypeTogether,
		ClientTypeFireworks,
		ClientTypePerplexity:
		return true
	default:
		return false
	}
}

// SelectMemoryModel selects a chat model for memory operations.
func SelectMemoryModel(ctx context.Context, modelsService *Service, queries *sqlc.Queries) (GetResponse, sqlc.LlmProvider, error) {
	if modelsService == nil {
		return GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("models service not configured")
	}
	candidates, err := modelsService.ListByType(ctx, ModelTypeChat)
	if err != nil || len(candidates) == 0 {
		return GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("no chat models available for memory operations")
	}
	selected := candidates[0]
	provider, err := FetchProviderByID(ctx, queries, selected.LlmProviderID)
	if err != nil {
		return GetResponse{}, sqlc.LlmProvider{}, err
	}
	return selected, provider, nil
}

// FetchProviderByID fetches a provider by ID.
func FetchProviderByID(ctx context.Context, queries *sqlc.Queries, providerID string) (sqlc.LlmProvider, error) {
	if strings.TrimSpace(providerID) == "" {
		return sqlc.LlmProvider{}, fmt.Errorf("provider id missing")
	}
	parsed, err := db.ParseUUID(providerID)
	if err != nil {
		return sqlc.LlmProvider{}, err
	}
	return queries.GetLlmProviderByID(ctx, parsed)
}
