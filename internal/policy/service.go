package policy

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/settings"
)

type Decision struct {
	BotID               string
	BotType             string
	AllowGuest          bool
	GroupRequireMention bool
}

type Service struct {
	bots     *bots.Service
	settings *settings.Service
	logger   *slog.Logger
}

func NewService(log *slog.Logger, botsService *bots.Service, settingsService *settings.Service) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		bots:     botsService,
		settings: settingsService,
		logger:   log.With(slog.String("service", "policy")),
	}
}

// Resolve evaluates the full access policy for a bot.
func (s *Service) Resolve(ctx context.Context, botID string) (Decision, error) {
	if s == nil || s.bots == nil || s.settings == nil {
		return Decision{}, fmt.Errorf("policy service not configured")
	}
	botID = strings.TrimSpace(botID)
	if botID == "" {
		return Decision{}, fmt.Errorf("bot id is required")
	}
	bot, err := s.bots.Get(ctx, botID)
	if err != nil {
		return Decision{}, err
	}
	botSettings, err := s.settings.GetBot(ctx, botID)
	if err != nil {
		return Decision{}, err
	}
	decision := Decision{
		BotID:               botID,
		BotType:             strings.TrimSpace(bot.Type),
		AllowGuest:          botSettings.AllowGuest,
		GroupRequireMention: botSettings.GroupRequireMention,
	}
	if decision.BotType == bots.BotTypePersonal {
		decision.AllowGuest = false
	}
	return decision, nil
}

// AllowGuest checks if the bot allows guest access. Implements router.PolicyService.
func (s *Service) AllowGuest(ctx context.Context, botID string) (bool, error) {
	decision, err := s.Resolve(ctx, botID)
	if err != nil {
		return false, err
	}
	return decision.AllowGuest, nil
}

// BotType returns the normalized bot type. Implements router.PolicyService.
func (s *Service) BotType(ctx context.Context, botID string) (string, error) {
	decision, err := s.Resolve(ctx, botID)
	if err != nil {
		return "", err
	}
	return decision.BotType, nil
}

// GroupRequireMention checks if the bot requires @mention in group chats. Implements router.PolicyService.
func (s *Service) GroupRequireMention(ctx context.Context, botID string) (bool, error) {
	decision, err := s.Resolve(ctx, botID)
	if err != nil {
		return true, err
	}
	return decision.GroupRequireMention, nil
}

// BotOwnerUserID returns bot owner's user id. Implements router.PolicyService.
func (s *Service) BotOwnerUserID(ctx context.Context, botID string) (string, error) {
	if s == nil || s.bots == nil {
		return "", fmt.Errorf("policy service not configured")
	}
	bot, err := s.bots.Get(ctx, strings.TrimSpace(botID))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(bot.OwnerUserID), nil
}
