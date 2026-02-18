package processlog

import (
	"time"
)

// ProcessLogStep represents a step in the conversation flow
type ProcessLogStep string

const (
	StepUserMessageReceived  ProcessLogStep = "user_message_received"
	StepHistoryLoaded        ProcessLogStep = "history_loaded"
	StepMemorySearched       ProcessLogStep = "memory_searched"
	StepMemoryLoaded         ProcessLogStep = "memory_loaded"
	StepPromptBuilt          ProcessLogStep = "prompt_built"
	StepLLMRequestSent       ProcessLogStep = "llm_request_sent"
	StepLLMResponseReceived  ProcessLogStep = "llm_response_received"
	StepResponseSent         ProcessLogStep = "response_sent"
	StepMemoryStored         ProcessLogStep = "memory_stored"
	StepStreamStarted        ProcessLogStep = "stream_started"
	StepStreamCompleted      ProcessLogStep = "stream_completed"
	StepStreamError          ProcessLogStep = "stream_error"
)

// ProcessLogLevel represents the log level
type ProcessLogLevel string

const (
	LevelDebug ProcessLogLevel = "debug"
	LevelInfo  ProcessLogLevel = "info"
	LevelWarn  ProcessLogLevel = "warn"
	LevelError ProcessLogLevel = "error"
)

// ProcessLog represents a single process log entry
type ProcessLog struct {
	ID         string           `json:"id"`
	BotID      string           `json:"bot_id"`
	ChatID     string           `json:"chat_id"`
	TraceID    string           `json:"trace_id"`
	UserID     string           `json:"user_id,omitempty"`
	Channel    string           `json:"channel,omitempty"`
	Step       ProcessLogStep   `json:"step"`
	Level      ProcessLogLevel  `json:"level"`
	Message    string           `json:"message,omitempty"`
	Data       map[string]any   `json:"data,omitempty"`
	DurationMs int              `json:"duration_ms,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

// ProcessLogStats represents statistics for a step
type ProcessLogStats struct {
	Step          ProcessLogStep `json:"step"`
	Count         int            `json:"count"`
	AvgDurationMs float64        `json:"avg_duration_ms"`
}

// ListRequest represents a request to list process logs
type ListRequest struct {
	BotID     string `json:"bot_id"`
	ChatID    string `json:"chat_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	Step      string `json:"step,omitempty"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

// CreateRequest represents a request to create a process log
type CreateRequest struct {
	BotID      string          `json:"bot_id"`
	ChatID     string          `json:"chat_id"`
	TraceID    string          `json:"trace_id,omitempty"`
	UserID     string          `json:"user_id,omitempty"`
	Channel    string          `json:"channel,omitempty"`
	Step       ProcessLogStep  `json:"step"`
	Level      ProcessLogLevel `json:"level,omitempty"`
	Message    string          `json:"message,omitempty"`
	Data       map[string]any  `json:"data,omitempty"`
	DurationMs int             `json:"duration_ms,omitempty"`
}
