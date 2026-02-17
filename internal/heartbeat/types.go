package heartbeat

import (
	"context"
	"time"
)

// EventTrigger defines which event types can wake up a heartbeat.
type EventTrigger string

const (
	// TriggerMessageCreated fires when a new message is stored for the bot.
	TriggerMessageCreated EventTrigger = "message_created"
	// TriggerScheduleCompleted fires after a scheduled task finishes.
	TriggerScheduleCompleted EventTrigger = "schedule_completed"
)

// AllTriggers is the set of recognised event triggers.
var AllTriggers = []EventTrigger{TriggerMessageCreated, TriggerScheduleCompleted}

// Config is the per-bot heartbeat configuration stored in the database.
type Config struct {
	ID              string         `json:"id"`
	BotID           string         `json:"bot_id"`
	Enabled         bool           `json:"enabled"`
	IntervalSeconds int            `json:"interval_seconds"`
	Prompt          string         `json:"prompt"`
	EventTriggers   []EventTrigger `json:"event_triggers"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// CreateRequest is the payload for creating a heartbeat config.
type CreateRequest struct {
	Enabled         *bool          `json:"enabled,omitempty"`
	IntervalSeconds int            `json:"interval_seconds"`
	Prompt          string         `json:"prompt"`
	EventTriggers   []EventTrigger `json:"event_triggers"`
}

// UpdateRequest is the payload for updating a heartbeat config.
type UpdateRequest struct {
	Enabled         *bool          `json:"enabled,omitempty"`
	IntervalSeconds *int           `json:"interval_seconds,omitempty"`
	Prompt          *string        `json:"prompt,omitempty"`
	EventTriggers   []EventTrigger `json:"event_triggers,omitempty"`
}

// ListResponse wraps a list of heartbeat configs.
type ListResponse struct {
	Items []Config `json:"items"`
}

// Triggerer triggers a heartbeat execution through the conversation flow.
type Triggerer interface {
	TriggerHeartbeat(ctx context.Context, botID string, payload TriggerPayload, token string) error
}

// TriggerPayload describes the parameters passed to the agent when a heartbeat fires.
type TriggerPayload struct {
	HeartbeatID string `json:"heartbeat_id"`
	Prompt      string `json:"prompt"`
	Reason      string `json:"reason"` // "periodic", event trigger name, or "manual"
	OwnerUserID string `json:"owner_user_id"`
}

// EvolutionReflectionPrompt is the default prompt used for self-evolution heartbeat.
const EvolutionReflectionPrompt = `Perform your self-evolution reflection cycle:
1. Re-read EXPERIMENTS.md, IDENTITY.md, SOUL.md, TOOLS.md
2. Reflect on recent conversations — what went well, what was slow or brittle
3. Record any new learnings at the top of EXPERIMENTS.md using the format:
   ### [today's date] Title
   **Goal**: ...
   **Method**: ...
   **Result**: ✅ Worked / ❌ Failed / ⚠️ Partial
   **Takeaway**: ...
4. If you discovered user preferences, update IDENTITY.md
5. If you found better workflows, update TOOLS.md
6. Summarize what you changed (or "no changes needed")`

// DefaultEvolutionIntervalSeconds is the default interval for the evolution heartbeat (24 hours).
const DefaultEvolutionIntervalSeconds = 86400

// EvolutionPromptMarker is a prefix used to identify system-created evolution heartbeats.
const EvolutionPromptMarker = "[evolution-reflection]"
