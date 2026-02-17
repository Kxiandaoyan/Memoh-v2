package heartbeat

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Kxiandaoyan/Memoh-v2/internal/automation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/boot"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	msgEvent "github.com/Kxiandaoyan/Memoh-v2/internal/message/event"
)

// Engine manages periodic and event-driven heartbeats for all bots.
type Engine struct {
	queries   *sqlc.Queries
	triggerer Triggerer
	hub       *msgEvent.Hub
	pool      *automation.CronPool
	jwtSecret string
	logger    *slog.Logger

	mu      sync.Mutex
	cancels map[string]func() // config ID → event subscription cancel
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewEngine creates a new heartbeat engine.
func NewEngine(
	log *slog.Logger,
	queries *sqlc.Queries,
	triggerer Triggerer,
	hub *msgEvent.Hub,
	pool *automation.CronPool,
	runtimeConfig *boot.RuntimeConfig,
) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		queries:   queries,
		triggerer: triggerer,
		hub:       hub,
		pool:      pool,
		jwtSecret: runtimeConfig.JwtSecret,
		logger:    log.With(slog.String("service", "heartbeat")),
		cancels:   map[string]func(){},
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Bootstrap loads all enabled heartbeat configs and starts their timers/subscriptions.
func (e *Engine) Bootstrap(ctx context.Context) error {
	if e.queries == nil {
		return fmt.Errorf("heartbeat queries not configured")
	}
	rows, err := e.queries.ListEnabledHeartbeatConfigs(ctx)
	if err != nil {
		return fmt.Errorf("list enabled heartbeat configs: %w", err)
	}
	for _, row := range rows {
		cfg := toConfig(row)
		e.startConfig(cfg)
	}
	e.logger.Info("heartbeat engine bootstrapped", slog.Int("configs", len(rows)))
	return nil
}

// Stop shuts down all periodic jobs and event subscriptions.
func (e *Engine) Stop() {
	e.cancel()
	e.mu.Lock()
	defer e.mu.Unlock()
	for id, cancelFn := range e.cancels {
		cancelFn()
		delete(e.cancels, id)
	}
}

// ── CRUD ──────────────────────────────────────────────────────────────

// Create creates a new heartbeat config for a bot.
func (e *Engine) Create(ctx context.Context, botID string, req CreateRequest) (Config, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return Config{}, fmt.Errorf("prompt is required")
	}
	if req.IntervalSeconds < 0 {
		return Config{}, fmt.Errorf("interval_seconds must be >= 0")
	}
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return Config{}, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	triggers := marshalTriggers(req.EventTriggers)

	row, err := e.queries.CreateHeartbeatConfig(ctx, sqlc.CreateHeartbeatConfigParams{
		BotID:           pgBotID,
		Enabled:         enabled,
		IntervalSeconds: int32(req.IntervalSeconds),
		Prompt:          req.Prompt,
		EventTriggers:   triggers,
	})
	if err != nil {
		return Config{}, err
	}
	cfg := toConfig(row)
	if cfg.Enabled {
		e.startConfig(cfg)
	}
	return cfg, nil
}

// Get returns a heartbeat config by ID.
func (e *Engine) Get(ctx context.Context, id string) (Config, error) {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return Config{}, err
	}
	row, err := e.queries.GetHeartbeatConfig(ctx, pgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Config{}, fmt.Errorf("heartbeat config not found")
		}
		return Config{}, err
	}
	return toConfig(row), nil
}

// List returns all heartbeat configs for a bot.
func (e *Engine) List(ctx context.Context, botID string) ([]Config, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := e.queries.ListHeartbeatConfigsByBot(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	items := make([]Config, 0, len(rows))
	for _, row := range rows {
		items = append(items, toConfig(row))
	}
	return items, nil
}

// Update modifies an existing heartbeat config.
func (e *Engine) Update(ctx context.Context, id string, req UpdateRequest) (Config, error) {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return Config{}, err
	}
	existing, err := e.queries.GetHeartbeatConfig(ctx, pgID)
	if err != nil {
		return Config{}, err
	}

	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	intervalSeconds := existing.IntervalSeconds
	if req.IntervalSeconds != nil {
		intervalSeconds = int32(*req.IntervalSeconds)
	}
	prompt := existing.Prompt
	if req.Prompt != nil {
		prompt = *req.Prompt
	}
	triggers := existing.EventTriggers
	if req.EventTriggers != nil {
		triggers = marshalTriggers(req.EventTriggers)
	}

	updated, err := e.queries.UpdateHeartbeatConfig(ctx, sqlc.UpdateHeartbeatConfigParams{
		ID:              pgID,
		Enabled:         enabled,
		IntervalSeconds: intervalSeconds,
		Prompt:          prompt,
		EventTriggers:   triggers,
	})
	if err != nil {
		return Config{}, err
	}
	cfg := toConfig(updated)
	e.restartConfig(cfg)
	return cfg, nil
}

// Delete removes a heartbeat config and stops its timer/subscriptions.
func (e *Engine) Delete(ctx context.Context, id string) error {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return err
	}
	if err := e.queries.DeleteHeartbeatConfig(ctx, pgID); err != nil {
		return err
	}
	e.stopConfig(id)
	return nil
}

// Fire manually triggers a heartbeat config by its ID.
func (e *Engine) Fire(ctx context.Context, configID, reason string) error {
	cfg, err := e.Get(ctx, configID)
	if err != nil {
		return err
	}
	return e.fire(ctx, cfg, reason)
}

// SeedEvolutionConfig creates (or enables) the system evolution heartbeat for a bot.
// It is called when allow_self_evolution is turned on.
func (e *Engine) SeedEvolutionConfig(ctx context.Context, botID string) error {
	existing, err := e.List(ctx, botID)
	if err != nil {
		return fmt.Errorf("list heartbeat configs: %w", err)
	}
	for _, cfg := range existing {
		if strings.Contains(cfg.Prompt, EvolutionPromptMarker) {
			if !cfg.Enabled {
				enabled := true
				_, err := e.Update(ctx, cfg.ID, UpdateRequest{Enabled: &enabled})
				return err
			}
			return nil
		}
	}
	prompt := EvolutionPromptMarker + "\n" + EvolutionReflectionPrompt
	enabled := true
	_, err = e.Create(ctx, botID, CreateRequest{
		Enabled:         &enabled,
		IntervalSeconds: DefaultEvolutionIntervalSeconds,
		Prompt:          prompt,
		EventTriggers:   nil,
	})
	return err
}

// DisableEvolutionConfig disables the system evolution heartbeat for a bot.
func (e *Engine) DisableEvolutionConfig(ctx context.Context, botID string) error {
	existing, err := e.List(ctx, botID)
	if err != nil {
		return fmt.Errorf("list heartbeat configs: %w", err)
	}
	for _, cfg := range existing {
		if strings.Contains(cfg.Prompt, EvolutionPromptMarker) && cfg.Enabled {
			enabled := false
			_, err := e.Update(ctx, cfg.ID, UpdateRequest{Enabled: &enabled})
			return err
		}
	}
	return nil
}

// ── Internal lifecycle management ─────────────────────────────────────

func (e *Engine) startConfig(cfg Config) {
	// Periodic scheduling via shared CronPool (@every Ns pattern).
	if cfg.IntervalSeconds > 0 && e.pool != nil {
		pattern := "@every " + strconv.Itoa(cfg.IntervalSeconds) + "s"
		cfgCopy := cfg
		if err := e.pool.Add(cfg.ID, pattern, func() {
			e.onPeriodicTick(cfgCopy)
		}); err != nil {
			e.logger.Error("failed to register heartbeat cron job",
				slog.String("config_id", cfg.ID),
				slog.String("pattern", pattern),
				slog.Any("error", err),
			)
		}
	}

	// Event subscriptions
	if len(cfg.EventTriggers) > 0 && e.hub != nil {
		_, ch, cancelSub := e.hub.Subscribe(cfg.BotID, msgEvent.DefaultBufferSize)
		e.mu.Lock()
		e.cancels[cfg.ID] = cancelSub
		e.mu.Unlock()
		go e.eventLoop(cfg, ch)
	}
}

func (e *Engine) stopConfig(id string) {
	// Remove periodic job from shared CronPool.
	if e.pool != nil {
		e.pool.Remove(id)
	}
	// Cancel event subscription.
	e.mu.Lock()
	defer e.mu.Unlock()
	if cancelFn, ok := e.cancels[id]; ok {
		cancelFn()
		delete(e.cancels, id)
	}
}

func (e *Engine) restartConfig(cfg Config) {
	e.stopConfig(cfg.ID)
	if cfg.Enabled {
		e.startConfig(cfg)
	}
}

// onPeriodicTick fires when the cron scheduler triggers the heartbeat interval.
func (e *Engine) onPeriodicTick(cfg Config) {
	select {
	case <-e.ctx.Done():
		return
	default:
	}

	e.logger.Debug("heartbeat periodic tick",
		slog.String("config_id", cfg.ID),
		slog.String("bot_id", cfg.BotID),
	)

	if err := e.fire(context.Background(), cfg, "periodic"); err != nil {
		e.logger.Error("heartbeat periodic trigger failed",
			slog.String("config_id", cfg.ID),
			slog.Any("error", err),
		)
	}
	// No manual reschedule needed — CronPool repeats automatically.
}

// eventLoop listens for events from the message hub and fires heartbeats.
func (e *Engine) eventLoop(cfg Config, ch <-chan msgEvent.Event) {
	triggerSet := make(map[EventTrigger]bool, len(cfg.EventTriggers))
	for _, t := range cfg.EventTriggers {
		triggerSet[t] = true
	}

	for {
		select {
		case <-e.ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			trigger := EventTrigger(evt.Type)
			if !triggerSet[trigger] {
				continue
			}
			e.logger.Debug("heartbeat event trigger",
				slog.String("config_id", cfg.ID),
				slog.String("bot_id", cfg.BotID),
				slog.String("event", string(evt.Type)),
			)
			if err := e.fire(context.Background(), cfg, string(trigger)); err != nil {
				e.logger.Error("heartbeat event trigger failed",
					slog.String("config_id", cfg.ID),
					slog.Any("error", err),
				)
			}
		}
	}
}

// fire executes the heartbeat by calling the triggerer.
func (e *Engine) fire(ctx context.Context, cfg Config, reason string) error {
	if e.triggerer == nil {
		return fmt.Errorf("heartbeat triggerer not configured")
	}
	ownerUserID, err := automation.ResolveBotOwner(ctx, e.queries, cfg.BotID)
	if err != nil {
		return fmt.Errorf("resolve bot owner: %w", err)
	}
	token, err := automation.GenerateTriggerToken(ownerUserID, e.jwtSecret, automation.DefaultTriggerTokenTTL)
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	payload := TriggerPayload{
		HeartbeatID: cfg.ID,
		Prompt:      cfg.Prompt,
		Reason:      reason,
		OwnerUserID: ownerUserID,
	}

	// If this is an evolution heartbeat, create an evolution log entry.
	if strings.Contains(cfg.Prompt, EvolutionPromptMarker) {
		pgBotID, parseErr := db.ParseUUID(cfg.BotID)
		if parseErr == nil {
			pgConfigID, _ := db.ParseUUID(cfg.ID)
			logRow, logErr := e.queries.CreateEvolutionLog(ctx, sqlc.CreateEvolutionLogParams{
				BotID:             pgBotID,
				HeartbeatConfigID: pgConfigID,
				TriggerReason:     reason,
			})
			if logErr != nil {
				e.logger.Warn("failed to create evolution log",
					slog.String("bot_id", cfg.BotID), slog.Any("error", logErr))
			} else {
				payload.EvolutionLogID = logRow.ID.String()
			}
		}
	}

	return e.triggerer.TriggerHeartbeat(ctx, cfg.BotID, payload, token)
}


// ── Helpers ───────────────────────────────────────────────────────────

func marshalTriggers(triggers []EventTrigger) []byte {
	if len(triggers) == 0 {
		return []byte("[]")
	}
	parts := make([]string, 0, len(triggers))
	for _, t := range triggers {
		parts = append(parts, `"`+string(t)+`"`)
	}
	return []byte("[" + strings.Join(parts, ",") + "]")
}

func parseTriggers(raw []byte) []EventTrigger {
	if len(raw) == 0 {
		return nil
	}
	s := strings.Trim(string(raw), "[]")
	if s == "" {
		return nil
	}
	var triggers []EventTrigger
	for _, part := range strings.Split(s, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part != "" {
			triggers = append(triggers, EventTrigger(part))
		}
	}
	return triggers
}

func toConfig(row sqlc.HeartbeatConfig) Config {
	cfg := Config{
		ID:              row.ID.String(),
		BotID:           row.BotID.String(),
		Enabled:         row.Enabled,
		IntervalSeconds: int(row.IntervalSeconds),
		Prompt:          row.Prompt,
		EventTriggers:   parseTriggers(row.EventTriggers),
	}
	if row.CreatedAt.Valid {
		cfg.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		cfg.UpdatedAt = row.UpdatedAt.Time
	}
	return cfg
}

// Ensure pgtype.UUID is used to suppress unused import warnings.
var _ pgtype.UUID
