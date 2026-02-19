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
	StepToolCallStarted      ProcessLogStep = "tool_call_started"
	StepToolCallCompleted    ProcessLogStep = "tool_call_completed"
	StepResponseSent         ProcessLogStep = "response_sent"
	StepMemoryStored         ProcessLogStep = "memory_stored"
	StepStreamStarted        ProcessLogStep = "stream_started"
	StepStreamCompleted      ProcessLogStep = "stream_completed"
	StepStreamError          ProcessLogStep = "stream_error"

	StepMemoryExtractStarted   ProcessLogStep = "memory_extract_started"
	StepMemoryExtractCompleted ProcessLogStep = "memory_extract_completed"
	StepMemoryExtractFailed    ProcessLogStep = "memory_extract_failed"
	StepTokenTrimmed           ProcessLogStep = "token_trimmed"
	StepSummaryLoaded          ProcessLogStep = "summary_loaded"
	StepSummaryRequested       ProcessLogStep = "summary_requested"
	StepSkillsLoaded           ProcessLogStep = "skills_loaded"
	StepOpenVikingContext          ProcessLogStep = "openviking_context"
	StepOpenVikingSession          ProcessLogStep = "openviking_session"
	StepOpenVikingSessionCompleted ProcessLogStep = "openviking_session_completed"
	StepOpenVikingSessionFailed    ProcessLogStep = "openviking_session_failed"
	StepEvolutionStarted           ProcessLogStep = "evolution_started"
	StepEvolutionCompleted     ProcessLogStep = "evolution_completed"
	StepEvolutionFailed        ProcessLogStep = "evolution_failed"

	StepMemoryFiltered         ProcessLogStep = "memory_filtered"
	StepQueryExpanded          ProcessLogStep = "query_expanded"
	StepTokenBudgetCalculated  ProcessLogStep = "token_budget_calculated"
	StepToolResultTrimmed      ProcessLogStep = "tool_result_trimmed"
	StepModelFallback          ProcessLogStep = "model_fallback"
	StepSkillsFiltered         ProcessLogStep = "skills_filtered"
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

// TraceExport is a self-contained diagnostic report for a single conversation round.
type TraceExport struct {
	Version        string            `json:"version"`
	ExportedAt     time.Time         `json:"exported_at"`
	TraceID        string            `json:"trace_id"`
	BotID          string            `json:"bot_id"`
	ChatID         string            `json:"chat_id"`
	Channel        string            `json:"channel,omitempty"`
	TimeRange      TraceTimeRange    `json:"time_range"`
	TotalDurationMs int              `json:"total_duration_ms"`
	Summary        TraceSummary      `json:"summary"`
	Steps          []TraceExportStep `json:"steps"`
}

type TraceTimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type TraceSummary struct {
	UserQuery         string         `json:"user_query"`
	AssistantResponse string         `json:"assistant_response,omitempty"`
	Model             string         `json:"model,omitempty"`
	Provider          string         `json:"provider,omitempty"`
	TokenUsage        map[string]any `json:"token_usage,omitempty"`
	StepsCount        int            `json:"steps_count"`
	Errors            []string       `json:"errors,omitempty"`
	Warnings          []string       `json:"warnings,omitempty"`
}

type TraceExportStep struct {
	Step       ProcessLogStep  `json:"step"`
	Level      ProcessLogLevel `json:"level"`
	Message    string          `json:"message,omitempty"`
	Data       map[string]any  `json:"data,omitempty"`
	DurationMs int             `json:"duration_ms,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ChatExport aggregates all traces for a single chat session.
type ChatExport struct {
	Version        string        `json:"version"`
	ExportedAt     time.Time     `json:"exported_at"`
	BotID          string        `json:"bot_id"`
	ChatID         string        `json:"chat_id"`
	Channel        string        `json:"channel,omitempty"`
	TotalRounds    int           `json:"total_rounds"`
	TimeRange      TraceTimeRange `json:"time_range"`
	TotalDurationMs int          `json:"total_duration_ms"`
	Rounds         []TraceExport `json:"rounds"`
}
