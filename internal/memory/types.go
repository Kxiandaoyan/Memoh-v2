package memory

import "context"

type ctxKeyPreferredModel struct{}

// WithPreferredModel returns a context that carries a bot-specific memory model ID.
// The lazyLLMClient checks this value to select the right model instead of the
// global first-available fallback.
func WithPreferredModel(ctx context.Context, modelID string) context.Context {
	return context.WithValue(ctx, ctxKeyPreferredModel{}, modelID)
}

// PreferredModelFromCtx extracts the preferred memory model ID from context.
func PreferredModelFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyPreferredModel{}).(string); ok {
		return v
	}
	return ""
}

// LLM is the interface for LLM operations needed by memory service
type LLM interface {
	Extract(ctx context.Context, req ExtractRequest) (ExtractResponse, error)
	Decide(ctx context.Context, req DecideRequest) (DecideResponse, error)
	Compact(ctx context.Context, req CompactRequest) (CompactResponse, error)
	DetectLanguage(ctx context.Context, text string) (string, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AddRequest struct {
	Message          string         `json:"message,omitempty"`
	Messages         []Message      `json:"messages,omitempty"`
	BotID            string         `json:"bot_id,omitempty"`
	AgentID          string         `json:"agent_id,omitempty"`
	RunID            string         `json:"run_id,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	Filters          map[string]any `json:"filters,omitempty"`
	Infer            *bool          `json:"infer,omitempty"`
	EmbeddingEnabled *bool          `json:"embedding_enabled,omitempty"`
}

type SearchRequest struct {
	Query            string         `json:"query"`
	BotID            string         `json:"bot_id,omitempty"`
	AgentID          string         `json:"agent_id,omitempty"`
	RunID            string         `json:"run_id,omitempty"`
	Limit            int            `json:"limit,omitempty"`
	Filters          map[string]any `json:"filters,omitempty"`
	Sources          []string       `json:"sources,omitempty"`
	EmbeddingEnabled *bool          `json:"embedding_enabled,omitempty"`
	NoStats          bool           `json:"no_stats,omitempty"`
	// MMR reranking: retrieve OverfetchFactor×Limit candidates, then apply
	// Maximal Marginal Relevance to balance relevance and diversity.
	UseMMR         bool    `json:"use_mmr,omitempty"`
	MMRLambda      float64 `json:"mmr_lambda,omitempty"`       // 0.0=max diversity, 1.0=max relevance; default 0.7
	OverfetchRatio int     `json:"overfetch_ratio,omitempty"`  // fetch OverfetchRatio×Limit before MMR; default 3
	// Temporal decay: reduce scores for older memories exponentially.
	UseTemporalDecay  bool    `json:"use_temporal_decay,omitempty"`
	DecayHalfLifeDays float64 `json:"decay_half_life_days,omitempty"` // default 30
}

type UpdateRequest struct {
	MemoryID         string `json:"memory_id"`
	Memory           string `json:"memory"`
	EmbeddingEnabled *bool  `json:"embedding_enabled,omitempty"`
}

type GetAllRequest struct {
	BotID   string         `json:"bot_id,omitempty"`
	AgentID string         `json:"agent_id,omitempty"`
	RunID   string         `json:"run_id,omitempty"`
	Limit   int            `json:"limit,omitempty"`
	Filters map[string]any `json:"filters,omitempty"`
	NoStats bool           `json:"no_stats,omitempty"`
}

type DeleteAllRequest struct {
	BotID   string         `json:"bot_id,omitempty"`
	AgentID string         `json:"agent_id,omitempty"`
	RunID   string         `json:"run_id,omitempty"`
	Filters map[string]any `json:"filters,omitempty"`
}

type EmbedInput struct {
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	VideoURL string `json:"video_url,omitempty"`
}

type EmbedUpsertRequest struct {
	Type     string         `json:"type"`
	Provider string         `json:"provider,omitempty"`
	Model    string         `json:"model,omitempty"`
	Input    EmbedInput     `json:"input"`
	Source   string         `json:"source,omitempty"`
	BotID    string         `json:"bot_id,omitempty"`
	AgentID  string         `json:"agent_id,omitempty"`
	RunID    string         `json:"run_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Filters  map[string]any `json:"filters,omitempty"`
}

type EmbedUpsertResponse struct {
	Item       MemoryItem `json:"item"`
	Provider   string     `json:"provider"`
	Model      string     `json:"model"`
	Dimensions int        `json:"dimensions"`
}

type MemoryItem struct {
	ID          string         `json:"id"`
	Memory      string         `json:"memory"`
	Hash        string         `json:"hash,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	UpdatedAt   string         `json:"updated_at,omitempty"`
	Score       float64        `json:"score,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	BotID       string         `json:"bot_id,omitempty"`
	AgentID     string         `json:"agent_id,omitempty"`
	RunID       string         `json:"run_id,omitempty"`
	TopKBuckets []TopKBucket   `json:"top_k_buckets,omitempty"`
	CDFCurve    []CDFPoint     `json:"cdf_curve,omitempty"`
}

// TopKBucket represents one bar in the Top-K sparse dimension bar chart.
type TopKBucket struct {
	Index uint32  `json:"index"` // sparse dimension index (term hash)
	Value float32 `json:"value"` // weight (term frequency)
}

// CDFPoint represents one point on the cumulative contribution curve.
type CDFPoint struct {
	K          int     `json:"k"`          // rank position (1-based, sorted by value desc)
	Cumulative float64 `json:"cumulative"` // cumulative weight fraction [0.0, 1.0]
}

type SearchResponse struct {
	Results   []MemoryItem `json:"results"`
	Relations []any        `json:"relations,omitempty"`
}

type DeleteResponse struct {
	Message string `json:"message"`
}

type ExtractRequest struct {
	Messages []Message      `json:"messages"`
	Filters  map[string]any `json:"filters,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ExtractResponse struct {
	Facts []string `json:"facts"`
}

type CandidateMemory struct {
	ID        string         `json:"id"`
	Memory    string         `json:"memory"`
	CreatedAt string         `json:"created_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type DecideRequest struct {
	Facts      []string          `json:"facts"`
	Candidates []CandidateMemory `json:"candidates"`
	Filters    map[string]any    `json:"filters,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

type DecisionAction struct {
	Event     string `json:"event"`
	ID        string `json:"id,omitempty"`
	Text      string `json:"text"`
	OldMemory string `json:"old_memory,omitempty"`
}

type DecideResponse struct {
	Actions []DecisionAction `json:"actions"`
}

type CompactRequest struct {
	Memories    []CandidateMemory `json:"memories"`
	TargetCount int               `json:"target_count"`
	DecayDays   int               `json:"decay_days,omitempty"`
}

type CompactResponse struct {
	Facts []string `json:"facts"`
}

type CompactResult struct {
	BeforeCount int          `json:"before_count"`
	AfterCount  int          `json:"after_count"`
	Ratio       float64      `json:"ratio"`
	Results     []MemoryItem `json:"results"`
}

type UsageResponse struct {
	Count                 int   `json:"count"`
	TotalTextBytes        int64 `json:"total_text_bytes"`
	AvgTextBytes          int64 `json:"avg_text_bytes"`
	EstimatedStorageBytes int64 `json:"estimated_storage_bytes"`
}

type RebuildResult struct {
	FsCount       int `json:"fs_count"`
	QdrantCount   int `json:"qdrant_count"`
	MissingCount  int `json:"missing_count"`
	RestoredCount int `json:"restored_count"`
}
