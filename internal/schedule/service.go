package schedule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Kxiandaoyan/Memoh-v2/internal/automation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/boot"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	msgEvent "github.com/Kxiandaoyan/Memoh-v2/internal/message/event"
)

type Service struct {
	queries   *sqlc.Queries
	pool      *automation.CronPool
	triggerer Triggerer
	hub       *msgEvent.Hub
	jwtSecret string
	logger    *slog.Logger
}

func NewService(log *slog.Logger, queries *sqlc.Queries, triggerer Triggerer, pool *automation.CronPool, hub *msgEvent.Hub, runtimeConfig *boot.RuntimeConfig) *Service {
	return &Service{
		queries:   queries,
		pool:      pool,
		triggerer: triggerer,
		hub:       hub,
		jwtSecret: runtimeConfig.JwtSecret,
		logger:    log.With(slog.String("service", "schedule")),
	}
}

func (s *Service) Bootstrap(ctx context.Context) error {
	if s.queries == nil {
		return fmt.Errorf("schedule queries not configured")
	}
	items, err := s.queries.ListEnabledSchedules(ctx)
	if err != nil {
		return err
	}
	var failed int
	for _, item := range items {
		if err := s.scheduleJob(item); err != nil {
			failed++
			s.logger.Error("failed to register schedule, skipping",
				slog.String("schedule_id", item.ID.String()),
				slog.String("pattern", item.Pattern),
				slog.Any("error", err),
			)
		}
	}
	s.logger.Info("schedule bootstrap complete",
		slog.Int("total", len(items)),
		slog.Int("registered", len(items)-failed),
		slog.Int("failed", failed),
	)
	return nil
}

func (s *Service) Create(ctx context.Context, botID string, req CreateRequest) (Schedule, error) {
	if s.queries == nil {
		return Schedule{}, fmt.Errorf("schedule queries not configured")
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Description) == "" || strings.TrimSpace(req.Pattern) == "" || strings.TrimSpace(req.Command) == "" {
		return Schedule{}, fmt.Errorf("name, description, pattern, command are required")
	}
	if err := s.pool.ValidatePattern(req.Pattern); err != nil {
		return Schedule{}, fmt.Errorf("invalid cron pattern: %w", err)
	}
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return Schedule{}, err
	}
	maxCalls := pgtype.Int4{Valid: false}
	if req.MaxCalls.Set && req.MaxCalls.Value != nil {
		maxCalls = pgtype.Int4{Int32: int32(*req.MaxCalls.Value), Valid: true}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row, err := s.queries.CreateSchedule(ctx, sqlc.CreateScheduleParams{
		Name:        req.Name,
		Description: req.Description,
		Pattern:     req.Pattern,
		MaxCalls:    maxCalls,
		Enabled:     enabled,
		Command:     req.Command,
		BotID:       pgBotID,
		Platform:    req.Platform,
		ReplyTarget: req.ReplyTarget,
	})
	if err != nil {
		return Schedule{}, err
	}
	if row.Enabled {
		if err := s.scheduleJob(row); err != nil {
			return Schedule{}, err
		}
	}
	return toSchedule(row), nil
}

func (s *Service) Get(ctx context.Context, id string) (Schedule, error) {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return Schedule{}, err
	}
	row, err := s.queries.GetScheduleByID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Schedule{}, fmt.Errorf("schedule not found")
		}
		return Schedule{}, err
	}
	return toSchedule(row), nil
}

func (s *Service) List(ctx context.Context, botID string) ([]Schedule, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListSchedulesByBot(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	items := make([]Schedule, 0, len(rows))
	for _, row := range rows {
		items = append(items, toSchedule(row))
	}
	return items, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Schedule, error) {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return Schedule{}, err
	}
	existing, err := s.queries.GetScheduleByID(ctx, pgID)
	if err != nil {
		return Schedule{}, err
	}
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	description := existing.Description
	if req.Description != nil {
		description = *req.Description
	}
	pattern := existing.Pattern
	if req.Pattern != nil {
		if err := s.pool.ValidatePattern(*req.Pattern); err != nil {
			return Schedule{}, fmt.Errorf("invalid cron pattern: %w", err)
		}
		pattern = *req.Pattern
	}
	command := existing.Command
	if req.Command != nil {
		command = *req.Command
	}
	maxCalls := existing.MaxCalls
	if req.MaxCalls.Set {
		if req.MaxCalls.Value == nil {
			maxCalls = pgtype.Int4{Valid: false}
		} else {
			maxCalls = pgtype.Int4{Int32: int32(*req.MaxCalls.Value), Valid: true}
		}
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	updated, err := s.queries.UpdateSchedule(ctx, sqlc.UpdateScheduleParams{
		ID:          pgID,
		Name:        name,
		Description: description,
		Pattern:     pattern,
		MaxCalls:    maxCalls,
		Enabled:     enabled,
		Command:     command,
	})
	if err != nil {
		return Schedule{}, err
	}
	if err := s.rescheduleJob(updated); err != nil {
		return Schedule{}, fmt.Errorf("reschedule job: %w", err)
	}
	return toSchedule(updated), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteSchedule(ctx, pgID); err != nil {
		return err
	}
	s.pool.Remove(id)
	return nil
}

func (s *Service) Trigger(ctx context.Context, scheduleID string) error {
	if s.triggerer == nil {
		return fmt.Errorf("schedule triggerer not configured")
	}
	schedule, err := s.Get(ctx, scheduleID)
	if err != nil {
		return err
	}
	if !schedule.Enabled {
		return fmt.Errorf("schedule is disabled")
	}
	return s.runSchedule(ctx, schedule)
}

func (s *Service) runSchedule(ctx context.Context, schedule Schedule) error {
	if s.triggerer == nil {
		return fmt.Errorf("schedule triggerer not configured")
	}

	ownerUserID, err := automation.ResolveBotOwner(ctx, s.queries, schedule.BotID)
	if err != nil {
		return fmt.Errorf("resolve bot owner: %w", err)
	}

	token, err := automation.GenerateTriggerToken(ownerUserID, s.jwtSecret, automation.DefaultTriggerTokenTTL)
	if err != nil {
		return fmt.Errorf("generate trigger token: %w", err)
	}

	platform := schedule.Platform
	replyTarget := schedule.ReplyTarget
	if strings.TrimSpace(platform) == "" {
		if p, t := s.resolveDefaultRoute(ctx, schedule.BotID); p != "" {
			platform = p
			replyTarget = t
			s.logger.Info("runSchedule: resolved fallback channel from routes",
				slog.String("schedule_id", schedule.ID),
				slog.String("platform", platform),
				slog.String("reply_target", replyTarget),
			)
		}
	}

	err = s.triggerer.TriggerSchedule(ctx, schedule.BotID, TriggerPayload{
		ID:          schedule.ID,
		Name:        schedule.Name,
		Description: schedule.Description,
		Pattern:     schedule.Pattern,
		MaxCalls:    schedule.MaxCalls,
		Command:     schedule.Command,
		OwnerUserID: ownerUserID,
		Platform:    platform,
		ReplyTarget: replyTarget,
	}, token)
	if err != nil {
		return err
	}

	// Increment call count only after successful execution to avoid
	// consuming max_calls quota on failed attempts.
	updated, err := s.queries.IncrementScheduleCalls(ctx, toUUID(schedule.ID))
	if err != nil {
		s.logger.Warn("failed to increment schedule calls after execution",
			slog.String("schedule_id", schedule.ID),
			slog.Any("error", err),
		)
	} else if !updated.Enabled {
		s.pool.Remove(schedule.ID)
	}

	// Publish schedule_completed event so heartbeats with this trigger can fire.
	if s.hub != nil {
		s.hub.Publish(msgEvent.Event{
			Type:  msgEvent.EventTypeScheduleCompleted,
			BotID: schedule.BotID,
		})
	}
	return nil
}

func (s *Service) scheduleJob(schedule sqlc.Schedule) error {
	id := schedule.ID.String()
	if id == "" {
		return fmt.Errorf("schedule id missing")
	}
	job := func() {
		if err := s.runSchedule(context.Background(), toSchedule(schedule)); err != nil {
			s.logger.Error("scheduled job failed", slog.String("schedule_id", schedule.ID.String()), slog.Any("error", err))
		}
	}
	return s.pool.Add(id, schedule.Pattern, job)
}

func (s *Service) rescheduleJob(schedule sqlc.Schedule) error {
	id := schedule.ID.String()
	if id == "" {
		return nil
	}
	if !schedule.Enabled {
		s.pool.Remove(id)
		return nil
	}
	return s.scheduleJob(schedule)
}

func toSchedule(row sqlc.Schedule) Schedule {
	item := Schedule{
		ID:           row.ID.String(),
		Name:         row.Name,
		Description:  row.Description,
		Pattern:      row.Pattern,
		CurrentCalls: int(row.CurrentCalls),
		Enabled:      row.Enabled,
		Command:      row.Command,
		BotID:        row.BotID.String(),
		Platform:     row.Platform,
		ReplyTarget:  row.ReplyTarget,
	}
	if row.MaxCalls.Valid {
		max := int(row.MaxCalls.Int32)
		item.MaxCalls = &max
	}
	if row.CreatedAt.Valid {
		item.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	return item
}

// resolveDefaultRoute looks up the bot's channel routes and returns the
// platform and reply_target of the first (oldest) route. This provides a
// fallback when a schedule has no platform stored (e.g. created before the
// platform column was added, or via the web UI).
func (s *Service) resolveDefaultRoute(ctx context.Context, botID string) (platform, replyTarget string) {
	if s.queries == nil {
		return "", ""
	}
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return "", ""
	}
	routes, err := s.queries.ListChatRoutes(ctx, pgBotID)
	if err != nil || len(routes) == 0 {
		return "", ""
	}
	route := routes[0]
	return strings.TrimSpace(route.Platform), strings.TrimSpace(route.ReplyTarget.String)
}

func toUUID(id string) pgtype.UUID {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgID
}
