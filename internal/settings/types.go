package settings

const (
	DefaultMaxContextLoadTime = 24 * 60
	DefaultLanguage           = "auto"
	// Default history limits (in user turns)
	DefaultDMHistoryLimit       = 50      // DM conversations
	DefaultChannelHistoryLimit  = 30      // Channel/Group conversations
	DefaultEvolutionHistoryLimit = 50     // Evolution heartbeat (all conversation types)
)

type Settings struct {
	ChatModelID          string `json:"chat_model_id"`
	MemoryModelID        string `json:"memory_model_id"`
	EmbeddingModelID     string `json:"embedding_model_id"`
	VlmModelID           string `json:"vlm_model_id"`
	SearchProviderID     string `json:"search_provider_id"`
	MaxContextLoadTime   int    `json:"max_context_load_time"`
	DMHistoryLimit       int    `json:"dm_history_limit"`
	ChannelHistoryLimit  int    `json:"channel_history_limit"`
	Language             string `json:"language"`
	AllowGuest           bool   `json:"allow_guest"`
	GroupRequireMention  bool   `json:"group_require_mention"`
}

type UpsertRequest struct {
	ChatModelID         string `json:"chat_model_id,omitempty"`
	MemoryModelID       string `json:"memory_model_id,omitempty"`
	EmbeddingModelID    string `json:"embedding_model_id,omitempty"`
	VlmModelID          string `json:"vlm_model_id,omitempty"`
	SearchProviderID    string `json:"search_provider_id,omitempty"`
	MaxContextLoadTime  *int   `json:"max_context_load_time,omitempty"`
	Language            string `json:"language,omitempty"`
	AllowGuest          *bool  `json:"allow_guest,omitempty"`
	GroupRequireMention *bool  `json:"group_require_mention,omitempty"`
}
