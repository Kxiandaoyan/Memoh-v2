package preauth

import "time"

// Key represents a bot pre-authorization key.
type Key struct {
	ID             string    `json:"id"`
	BotID          string    `json:"bot_id"`
	Token          string    `json:"token"`
	IssuedByUserID string    `json:"issued_by_user_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	UsedAt         time.Time `json:"used_at"`
	CreatedAt      time.Time `json:"created_at"`
}
