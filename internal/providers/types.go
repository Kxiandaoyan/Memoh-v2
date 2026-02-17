package providers

import "time"

// ClientType represents the type of LLM provider client
type ClientType string

const (
	ClientTypeOpenAI          ClientType = "openai"
	ClientTypeOpenAICompat    ClientType = "openai-compat"
	ClientTypeAnthropic       ClientType = "anthropic"
	ClientTypeGoogle          ClientType = "google"
	ClientTypeAzure           ClientType = "azure"
	ClientTypeBedrock         ClientType = "bedrock"
	ClientTypeMistral         ClientType = "mistral"
	ClientTypeXAI             ClientType = "xai"
	ClientTypeOllama          ClientType = "ollama"
	ClientTypeDashscope       ClientType = "dashscope"
	ClientTypeDeepSeek        ClientType = "deepseek"
	ClientTypeZaiGlobal       ClientType = "zai-global"
	ClientTypeZaiCN           ClientType = "zai-cn"
	ClientTypeZaiCodingGlobal ClientType = "zai-coding-global"
	ClientTypeZaiCodingCN     ClientType = "zai-coding-cn"
	ClientTypeMinimaxGlobal   ClientType = "minimax-global"
	ClientTypeMinimaxCN       ClientType = "minimax-cn"
	ClientTypeMoonshotGlobal  ClientType = "moonshot-global"
	ClientTypeMoonshotCN      ClientType = "moonshot-cn"
	ClientTypeVolcengine      ClientType = "volcengine"
	ClientTypeVolcengineCoding ClientType = "volcengine-coding"
	ClientTypeQianfan         ClientType = "qianfan"
	ClientTypeGroq            ClientType = "groq"
	ClientTypeOpenRouter      ClientType = "openrouter"
	ClientTypeTogether        ClientType = "together"
	ClientTypeFireworks       ClientType = "fireworks"
	ClientTypePerplexity      ClientType = "perplexity"
)

// CreateRequest represents a request to create a new LLM provider
type CreateRequest struct {
	Name       string         `json:"name" validate:"required"`
	ClientType ClientType     `json:"client_type" validate:"required"`
	BaseURL    string         `json:"base_url" validate:"required,url"`
	APIKey     string         `json:"api_key"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// UpdateRequest represents a request to update an existing LLM provider
type UpdateRequest struct {
	Name       *string        `json:"name,omitempty"`
	ClientType *ClientType    `json:"client_type,omitempty"`
	BaseURL    *string        `json:"base_url,omitempty"`
	APIKey     *string        `json:"api_key,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// GetResponse represents the response for getting a provider
type GetResponse struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	ClientType string         `json:"client_type"`
	BaseURL    string         `json:"base_url"`
	APIKey     string         `json:"api_key,omitempty"` // masked in response
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// ListResponse represents the response for listing providers
type ListResponse struct {
	Providers []GetResponse `json:"providers"`
	Total     int64         `json:"total"`
}

// CountResponse represents the count response
type CountResponse struct {
	Count int64 `json:"count"`
}

// TestRequest represents a request to test provider connection
type TestRequest struct {
	ClientType ClientType `json:"client_type" validate:"required"`
	BaseURL    string     `json:"base_url" validate:"required,url"`
	APIKey     string     `json:"api_key"`
	Model      string     `json:"model"` // optional test model
}

// TestResponse represents the result of testing a provider
type TestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Latency int64  `json:"latency_ms,omitempty"` // latency in milliseconds
}
