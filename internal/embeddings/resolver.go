package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	"github.com/Kxiandaoyan/Memoh-v2/internal/models"
)

const (
	TypeText       = "text"
	TypeMultimodal = "multimodal"

	ProviderOpenAI    = "openai"
	ProviderBedrock   = "bedrock"
	ProviderDashScope = "dashscope"
)

type Request struct {
	Type       string
	Provider   string
	Model      string
	Dimensions int
	Input      Input
}

type Input struct {
	Text     string
	ImageURL string
	VideoURL string
}

type Usage struct {
	InputTokens int
	ImageTokens int
	Duration    int
}

type Result struct {
	Type       string
	Provider   string
	Model      string
	Dimensions int
	Embedding  []float32
	Usage      Usage
}

const (
	embeddingCacheMaxSize = 1024
	embeddingCacheTTL     = 24 * time.Hour
)

type embeddingCacheEntry struct {
	embedding []float32
	createdAt time.Time
}

type embeddingCache struct {
	mu      sync.RWMutex
	entries map[string]embeddingCacheEntry
}

func newEmbeddingCache() *embeddingCache {
	return &embeddingCache{entries: make(map[string]embeddingCacheEntry)}
}

func (c *embeddingCache) key(model, text string) string {
	h := sha256.Sum256([]byte(model + "|" + text))
	return hex.EncodeToString(h[:])
}

func (c *embeddingCache) get(model, text string) ([]float32, bool) {
	k := c.key(model, text)
	c.mu.RLock()
	entry, ok := c.entries[k]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(entry.createdAt) > embeddingCacheTTL {
		c.mu.Lock()
		delete(c.entries, k)
		c.mu.Unlock()
		return nil, false
	}
	return entry.embedding, true
}

func (c *embeddingCache) set(model, text string, embedding []float32) {
	k := c.key(model, text)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= embeddingCacheMaxSize {
		oldest := ""
		oldestTime := time.Now()
		for ek, ev := range c.entries {
			if ev.createdAt.Before(oldestTime) {
				oldest = ek
				oldestTime = ev.createdAt
			}
		}
		if oldest != "" {
			delete(c.entries, oldest)
		}
	}
	c.entries[k] = embeddingCacheEntry{embedding: embedding, createdAt: time.Now()}
}

type Resolver struct {
	modelsService *models.Service
	queries       *sqlc.Queries
	timeout       time.Duration
	logger        *slog.Logger
	cache         *embeddingCache
}

func NewResolver(log *slog.Logger, modelsService *models.Service, queries *sqlc.Queries, timeout time.Duration) *Resolver {
	return &Resolver{
		modelsService: modelsService,
		queries:       queries,
		timeout:       timeout,
		logger:        log.With(slog.String("service", "embeddings")),
		cache:         newEmbeddingCache(),
	}
}

func (r *Resolver) Embed(ctx context.Context, req Request) (Result, error) {
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.Model = strings.TrimSpace(req.Model)
	req.Input.Text = strings.TrimSpace(req.Input.Text)
	req.Input.ImageURL = strings.TrimSpace(req.Input.ImageURL)
	req.Input.VideoURL = strings.TrimSpace(req.Input.VideoURL)

	if req.Type == "" {
		return Result{}, errors.New("type is required")
	}
	switch req.Type {
	case TypeText:
		if req.Provider != "" && req.Provider != ProviderOpenAI {
			return Result{}, errors.New("invalid provider for text embeddings")
		}
		if req.Input.Text == "" {
			return Result{}, errors.New("text input is required")
		}
	case TypeMultimodal:
		if req.Provider != "" && req.Provider != ProviderBedrock && req.Provider != ProviderDashScope {
			return Result{}, errors.New("invalid provider for multimodal embeddings")
		}
		if req.Input.Text == "" && req.Input.ImageURL == "" && req.Input.VideoURL == "" {
			return Result{}, errors.New("multimodal input is required")
		}
	default:
		return Result{}, errors.New("invalid embeddings type")
	}

	selected, err := r.selectEmbeddingModel(ctx, req)
	if err != nil {
		return Result{}, err
	}
	provider, err := r.fetchProvider(ctx, selected.LlmProviderID)
	if err != nil {
		return Result{}, err
	}

	req.Model = selected.ModelID
	req.Dimensions = selected.Dimensions
	req.Provider = strings.ToLower(strings.TrimSpace(provider.ClientType))
	if req.Model == "" {
		return Result{}, errors.New("embedding model id not configured")
	}
	if req.Dimensions <= 0 {
		return Result{}, errors.New("embedding model dimensions not configured")
	}

	timeout := r.timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	switch req.Type {
	case TypeText:
		if req.Provider != ProviderOpenAI {
			return Result{}, errors.New("provider not implemented")
		}
		if cached, ok := r.cache.get(req.Model, req.Input.Text); ok {
			r.logger.Debug("embedding cache hit", slog.String("model", req.Model))
			return Result{
				Type:       req.Type,
				Provider:   req.Provider,
				Model:      req.Model,
				Dimensions: req.Dimensions,
				Embedding:  cached,
			}, nil
		}
		embedder, err := NewOpenAIEmbedder(r.logger, provider.ApiKey, provider.BaseUrl, req.Model, req.Dimensions, timeout)
		if err != nil {
			return Result{}, err
		}
		vector, err := embedder.Embed(ctx, req.Input.Text)
		if err != nil {
			return Result{}, err
		}
		r.cache.set(req.Model, req.Input.Text, vector)
		return Result{
			Type:       req.Type,
			Provider:   req.Provider,
			Model:      req.Model,
			Dimensions: req.Dimensions,
			Embedding:  vector,
		}, nil
	case TypeMultimodal:
		if req.Provider == ProviderDashScope {
			if strings.TrimSpace(provider.ApiKey) == "" {
				return Result{}, errors.New("dashscope api key is required")
			}
			dashscope := NewDashScopeEmbedder(r.logger, provider.ApiKey, provider.BaseUrl, req.Model, timeout)
			vector, usage, err := dashscope.Embed(ctx, req.Input.Text, req.Input.ImageURL, req.Input.VideoURL)
			if err != nil {
				return Result{}, err
			}
			return Result{
				Type:       req.Type,
				Provider:   req.Provider,
				Model:      req.Model,
				Dimensions: req.Dimensions,
				Embedding:  vector,
				Usage: Usage{
					InputTokens: usage.InputTokens,
					ImageTokens: usage.ImageTokens,
					Duration:    usage.Duration,
				},
			}, nil
		}
		return Result{}, errors.New("provider not implemented")
	default:
		return Result{}, errors.New("invalid embeddings type")
	}
}

func (r *Resolver) selectEmbeddingModel(ctx context.Context, req Request) (models.GetResponse, error) {
	if r.modelsService == nil {
		return models.GetResponse{}, errors.New("models service not configured")
	}

	var candidates []models.GetResponse
	var err error
	if req.Provider != "" {
		candidates, err = r.modelsService.ListByClientType(ctx, models.ClientType(req.Provider))
	} else {
		candidates, err = r.modelsService.ListByType(ctx, models.ModelTypeEmbedding)
	}
	if err != nil {
		return models.GetResponse{}, err
	}

	filtered := make([]models.GetResponse, 0, len(candidates))
	for _, model := range candidates {
		if model.Type != models.ModelTypeEmbedding {
			continue
		}
		if req.Type == TypeMultimodal && !model.IsMultimodal {
			continue
		}
		if req.Type == TypeText && model.IsMultimodal {
			continue
		}
		filtered = append(filtered, model)
	}
	if len(filtered) == 0 {
		return models.GetResponse{}, errors.New("no embedding models available")
	}
	if req.Model != "" {
		for _, model := range filtered {
			if model.ModelID == req.Model {
				return model, nil
			}
		}
		return models.GetResponse{}, errors.New("embedding model not found")
	}
	return filtered[0], nil
}

func (r *Resolver) fetchProvider(ctx context.Context, providerID string) (sqlc.LlmProvider, error) {
	if r.queries == nil {
		return sqlc.LlmProvider{}, errors.New("llm provider queries not configured")
	}
	if strings.TrimSpace(providerID) == "" {
		return sqlc.LlmProvider{}, errors.New("llm provider id missing")
	}
	parsed, err := uuid.Parse(providerID)
	if err != nil {
		return sqlc.LlmProvider{}, err
	}
	pgID := pgtype.UUID{Valid: true}
	copy(pgID.Bytes[:], parsed[:])
	return r.queries.GetLlmProviderByID(ctx, pgID)
}
