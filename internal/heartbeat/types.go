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
	HeartbeatID    string `json:"heartbeat_id"`
	Prompt         string `json:"prompt"`
	Reason         string `json:"reason"` // "periodic", event trigger name, or "manual"
	OwnerUserID    string `json:"owner_user_id"`
	EvolutionLogID string `json:"evolution_log_id,omitempty"` // set when this is an evolution heartbeat
}

// EvolutionReflectionPrompt is the default prompt used for self-evolution heartbeat.
// It implements a three-phase organic evolution cycle: Reflect → Experiment → Review.
// The cycle is conversation-driven: if there's nothing to learn from, no changes are made.
const EvolutionReflectionPrompt = `[evolution-reflection] Perform your organic self-evolution cycle.

IMPORTANT: This is not a forced exercise. Only make changes if your recent conversations
provide genuine material to learn from. If conversations have been few or uneventful,
it's perfectly fine to report "no evolution needed" and stop.

## Phase 1: REFLECT — Mine your conversations for signal

1. Re-read your current files: IDENTITY.md, SOUL.md, TOOLS.md, EXPERIMENTS.md, NOTES.md
2. Review your recent conversation history. Look for:
   - Friction: moments where you struggled, gave wrong answers, or frustrated the user
   - Delight: moments where you were especially helpful or the user expressed satisfaction
   - Patterns: recurring topics, repeated questions, emerging user preferences
   - Gaps: knowledge or capabilities you lacked when the user needed them
3. If you find nothing meaningful — no friction, no new patterns, no gaps — STOP HERE.
   Report "No evolution needed — recent conversations were handled well." and end.

## Phase 2: EXPERIMENT — Make targeted, small improvements

Only proceed if Phase 1 surfaced actionable insights. For each insight:

1. Log it in EXPERIMENTS.md using this format:
   ### [today's date] Brief descriptive title
   **Trigger**: What conversation/event prompted this?
   **Observation**: What did you notice?
   **Action**: What specific change are you making?
   **Expected outcome**: How will this improve future interactions?

2. Apply the change to the appropriate file:
   - User preferences or personality adjustments → IDENTITY.md
   - Behavioral rules or communication style → SOUL.md
   - Workflow improvements or tool usage notes → TOOLS.md

3. Keep changes SMALL and REVERSIBLE. One or two targeted edits per cycle.
   Never rewrite entire files. Evolution is incremental, not revolutionary.

## Phase 3: REVIEW — Self-healing and maintenance

1. Check scheduled tasks — have they been running? Any stale or failing tasks?
2. Review NOTES.md — distill important learnings into long-term files, trim noise
3. Verify coordination files (if /shared exists) — are your outputs current?
4. If anything looks broken or anomalous, flag it to the user

## Output

End with a brief summary:
- What you reflected on (or "nothing notable")
- What you changed (or "no changes needed")
- Any issues flagged for the user (or "all clear")`

// DefaultEvolutionIntervalSeconds is the default interval for the evolution heartbeat (24 hours).
const DefaultEvolutionIntervalSeconds = 86400

// EvolutionPromptMarker is a prefix used to identify system-created evolution heartbeats.
const EvolutionPromptMarker = "[evolution-reflection]"
