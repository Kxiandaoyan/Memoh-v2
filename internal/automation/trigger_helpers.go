package automation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kxiandaoyan/Memoh-v2/internal/auth"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
)

// DefaultTriggerTokenTTL is the default lifetime for trigger JWT tokens.
const DefaultTriggerTokenTTL = 10 * time.Minute

// ResolveBotOwner returns the owner user ID for the given bot.
// This is shared by schedule.Service and heartbeat.Engine.
func ResolveBotOwner(ctx context.Context, queries *sqlc.Queries, botID string) (string, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return "", err
	}
	bot, err := queries.GetBotByID(ctx, pgBotID)
	if err != nil {
		return "", fmt.Errorf("get bot: %w", err)
	}
	ownerID := bot.OwnerUserID.String()
	if ownerID == "" {
		return "", fmt.Errorf("bot owner not found")
	}
	return ownerID, nil
}

// GenerateTriggerToken creates a short-lived JWT for trigger callbacks.
// This is shared by schedule.Service and heartbeat.Engine.
func GenerateTriggerToken(userID, jwtSecret string, ttl time.Duration) (string, error) {
	if strings.TrimSpace(jwtSecret) == "" {
		return "", fmt.Errorf("jwt secret not configured")
	}
	signed, _, err := auth.GenerateToken(userID, jwtSecret, ttl)
	if err != nil {
		return "", err
	}
	return "Bearer " + signed, nil
}
