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
	Reason      string `json:"reason"` // "periodic" or event trigger name
	OwnerUserID string `json:"owner_user_id"`
}
