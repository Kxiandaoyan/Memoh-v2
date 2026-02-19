package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

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
	compactor MemoryCompactor
	dbPool    *pgxpool.Pool  // optional; enables active-hours column loading and evolution snapshots
	timezone  *time.Location // global timezone used for active-hours checks
	dataDir   string         // root data directory for bot persona files (set via SetDataDir)

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
		e.logger.Warn("heartbeat config create failed", slog.String("bot_id", botID), slog.Any("error", err))
		return Config{}, err
	}
	cfg := toConfig(row)
	hbType := "heartbeat"
	if strings.Contains(cfg.Prompt, EvolutionPromptMarker) {
		hbType = "evolution"
	} else if strings.Contains(cfg.Prompt, MemoryCompactPromptMarker) {
		hbType = "memory_compact"
	}
	e.logger.Info("heartbeat config created", slog.String("bot_id", botID), slog.String("type", hbType))
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
		e.logger.Warn("heartbeat config update failed", slog.String("config_id", id), slog.Any("error", err))
		return Config{}, err
	}
	cfg := toConfig(updated)
	e.logger.Info("heartbeat config updated", slog.String("config_id", id))
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
		e.logger.Warn("heartbeat config delete failed", slog.String("config_id", id), slog.Any("error", err))
		return err
	}
	e.logger.Info("heartbeat config deleted", slog.String("config_id", id))
	e.stopConfig(id)
	return nil
}

// Fire manually triggers a heartbeat config by its ID.
func (e *Engine) Fire(ctx context.Context, configID, reason string) error {
	cfg, err := e.Get(ctx, configID)
	if err != nil {
		e.logger.Warn("heartbeat fire: get config failed", slog.String("config_id", configID), slog.Any("error", err))
		return err
	}
	if err := e.fire(ctx, cfg, reason); err != nil {
		e.logger.Warn("heartbeat fire failed", slog.String("config_id", configID), slog.Any("error", err))
		return err
	}
	return nil
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

// SetMemoryCompactor registers a MemoryCompactor for direct memory compaction.
func (e *Engine) SetMemoryCompactor(c MemoryCompactor) {
	e.compactor = c
}

// SeedMemoryCompactConfig creates (or enables) the system memory compaction heartbeat for a bot.
func (e *Engine) SeedMemoryCompactConfig(ctx context.Context, botID string) error {
	existing, err := e.List(ctx, botID)
	if err != nil {
		e.logger.Warn("memory compact config seed failed", slog.String("bot_id", botID), slog.Any("error", err))
		return fmt.Errorf("list heartbeat configs: %w", err)
	}
	for _, cfg := range existing {
		if strings.Contains(cfg.Prompt, MemoryCompactPromptMarker) {
			if !cfg.Enabled {
				enabled := true
				_, err := e.Update(ctx, cfg.ID, UpdateRequest{Enabled: &enabled})
				if err != nil {
					e.logger.Warn("memory compact config seed failed", slog.String("bot_id", botID), slog.Any("error", err))
					return err
				}
			}
			e.logger.Info("memory compact config seeded", slog.String("bot_id", botID))
			return nil
		}
	}
	enabled := true
	_, err = e.Create(ctx, botID, CreateRequest{
		Enabled:         &enabled,
		IntervalSeconds: DefaultMemoryCompactIntervalSeconds,
		Prompt:          MemoryCompactPromptMarker + " Automatic memory compaction.",
		EventTriggers:   nil,
	})
	if err != nil {
		e.logger.Warn("memory compact config seed failed", slog.String("bot_id", botID), slog.Any("error", err))
		return err
	}
	e.logger.Info("memory compact config seeded", slog.String("bot_id", botID))
	return nil
}

// DisableMemoryCompactConfig disables the system memory compaction heartbeat for a bot.
func (e *Engine) DisableMemoryCompactConfig(ctx context.Context, botID string) error {
	existing, err := e.List(ctx, botID)
	if err != nil {
		e.logger.Warn("memory compact config disable failed", slog.String("bot_id", botID), slog.Any("error", err))
		return fmt.Errorf("list heartbeat configs: %w", err)
	}
	for _, cfg := range existing {
		if strings.Contains(cfg.Prompt, MemoryCompactPromptMarker) && cfg.Enabled {
			enabled := false
			_, err := e.Update(ctx, cfg.ID, UpdateRequest{Enabled: &enabled})
			if err != nil {
				e.logger.Warn("memory compact config disable failed", slog.String("bot_id", botID), slog.Any("error", err))
				return err
			}
			e.logger.Info("memory compact config disabled", slog.String("bot_id", botID))
			return nil
		}
	}
	e.logger.Info("memory compact config disabled", slog.String("bot_id", botID))
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
	// Skip firing if outside the configured active hours window.
	if !e.isWithinActiveHours(ctx, cfg) {
		e.logger.Debug("heartbeat skipped: outside active hours",
			slog.String("config_id", cfg.ID),
			slog.String("bot_id", cfg.BotID),
		)
		return nil
	}

	// Memory compaction heartbeats are handled directly, bypassing the conversation flow.
	if strings.Contains(cfg.Prompt, MemoryCompactPromptMarker) {
		return e.fireMemoryCompact(ctx, cfg, reason)
	}

	if e.triggerer == nil {
		return fmt.Errorf("heartbeat triggerer not configured")
	}
	ownerUserID, err := automation.ResolveBotOwner(ctx, e.queries, cfg.BotID)
	if err != nil {
		e.logger.Warn("heartbeat fire: resolve bot owner failed", slog.String("bot_id", cfg.BotID), slog.Any("error", err))
		return fmt.Errorf("resolve bot owner: %w", err)
	}
	token, err := automation.GenerateTriggerToken(ownerUserID, e.jwtSecret, automation.DefaultTriggerTokenTTL)
	if err != nil {
		e.logger.Warn("heartbeat fire: generate token failed", slog.String("bot_id", cfg.BotID), slog.Any("error", err))
		return fmt.Errorf("generate token: %w", err)
	}

	intervalPattern := ""
	if cfg.IntervalSeconds > 0 {
		intervalPattern = "@every " + strconv.Itoa(cfg.IntervalSeconds) + "s"
	}
	payload := TriggerPayload{
		HeartbeatID:     cfg.ID,
		Prompt:          cfg.Prompt,
		Reason:          reason,
		OwnerUserID:     ownerUserID,
		IntervalPattern: intervalPattern,
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
				// Snapshot persona files before evolution so we can roll back later.
				e.saveEvolutionSnapshot(ctx, logRow.ID.String(), cfg.BotID)
			}
		}
	}

	if err := e.triggerer.TriggerHeartbeat(ctx, cfg.BotID, payload, token); err != nil {
		e.logger.Warn("heartbeat fire: trigger failed", slog.String("bot_id", cfg.BotID), slog.Any("error", err))
		return err
	}
	return nil
}

// fireMemoryCompact directly invokes the memory compactor without going through the bot conversation flow.
func (e *Engine) fireMemoryCompact(ctx context.Context, cfg Config, reason string) error {
	if e.compactor == nil {
		e.logger.Warn("memory compactor not configured, skipping memory compaction heartbeat",
			slog.String("bot_id", cfg.BotID))
		return nil
	}
	e.logger.Info("memory compaction heartbeat fired",
		slog.String("bot_id", cfg.BotID),
		slog.String("reason", reason),
	)
	if err := e.compactor.CompactBot(ctx, cfg.BotID, DefaultMemoryCompactRatio, DefaultMemoryCompactMinCount); err != nil {
		e.logger.Error("memory compaction failed",
			slog.String("bot_id", cfg.BotID),
			slog.Any("error", err),
		)
		return err
	}
	e.logger.Info("memory compaction completed", slog.String("bot_id", cfg.BotID))
	return nil
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

// SetPool registers the database pool used to load active-hours configuration.
// This is optional; without it the active-hours check is skipped.
func (e *Engine) SetPool(pool *pgxpool.Pool) {
	e.dbPool = pool
}

// SetTimezone sets the timezone used for active-hours checks.
func (e *Engine) SetTimezone(loc *time.Location) {
	if loc != nil {
		e.timezone = loc
	}
}

// SetDataDir sets the host data directory root used to snapshot persona files
// before evolution runs. Pass the same value as mcp.DataRoot in config.
func (e *Engine) SetDataDir(dir string) {
	e.dataDir = strings.TrimSpace(dir)
}

// personaFileNames lists the files captured in an evolution snapshot.
var personaFileNames = []string{"IDENTITY.md", "SOUL.md", "TOOLS.md", "EXPERIMENTS.md", "NOTES.md"}

// snapshotPersonaFiles reads persona files from the bot's data directory and
// returns their contents as a JSON-serialisable map. Files that are missing or
// empty are omitted. Returns nil when the data directory is not configured.
func (e *Engine) snapshotPersonaFiles(botID string) map[string]string {
	if e.dataDir == "" {
		return nil
	}
	botDir := filepath.Join(e.dataDir, "bots", botID)
	snapshot := make(map[string]string)
	for _, name := range personaFileNames {
		content, err := os.ReadFile(filepath.Join(botDir, name))
		if err == nil && len(strings.TrimSpace(string(content))) > 0 {
			snapshot[name] = string(content)
		}
	}
	if len(snapshot) == 0 {
		return nil
	}
	return snapshot
}

// RollbackEvolution reads the files_snapshot from the given evolution log and
// writes each captured file back to the bot's data directory. Returns the list
// of file names that were restored.
func (e *Engine) RollbackEvolution(ctx context.Context, botID, logID string) ([]string, error) {
	if e.dbPool == nil {
		return nil, fmt.Errorf("database pool not configured")
	}
	if e.dataDir == "" {
		return nil, fmt.Errorf("data directory not configured (call SetDataDir)")
	}

	var rawSnapshot []byte
	err := e.dbPool.QueryRow(ctx,
		`SELECT files_snapshot FROM evolution_logs WHERE id=$1 AND bot_id=$2`,
		logID, botID,
	).Scan(&rawSnapshot)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("evolution log not found: %s", logID)
		}
		return nil, fmt.Errorf("query evolution log: %w", err)
	}
	if len(rawSnapshot) == 0 {
		return nil, fmt.Errorf("no snapshot available for this evolution log")
	}

	var snapshot map[string]string
	if err := json.Unmarshal(rawSnapshot, &snapshot); err != nil {
		return nil, fmt.Errorf("parse snapshot: %w", err)
	}
	if len(snapshot) == 0 {
		return nil, fmt.Errorf("no snapshot available for this evolution log")
	}

	botDir := filepath.Join(e.dataDir, "bots", botID)
	if err := os.MkdirAll(botDir, 0o755); err != nil {
		return nil, fmt.Errorf("create bot dir: %w", err)
	}

	var restored []string
	for name, content := range snapshot {
		fpath := filepath.Join(botDir, filepath.Base(name))
		if err := os.WriteFile(fpath, []byte(content), 0o644); err != nil {
			e.logger.Warn("rollback: failed to write file",
				slog.String("file", name), slog.Any("error", err))
			continue
		}
		restored = append(restored, name)
	}

	e.logger.Info("evolution rollback completed",
		slog.String("bot_id", botID),
		slog.String("log_id", logID),
		slog.Int("files_restored", len(restored)),
	)
	return restored, nil
}

// saveEvolutionSnapshot persists files_snapshot for the given evolution log ID.
func (e *Engine) saveEvolutionSnapshot(ctx context.Context, logID, botID string) {
	if e.dbPool == nil {
		return
	}
	snapshot := e.snapshotPersonaFiles(botID)
	if snapshot == nil {
		return
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		e.logger.Warn("evolution snapshot: marshal failed", slog.Any("error", err))
		return
	}
	if _, err := e.dbPool.Exec(ctx,
		`UPDATE evolution_logs SET files_snapshot=$1 WHERE id=$2`,
		raw, logID,
	); err != nil {
		e.logger.Warn("evolution snapshot: db update failed",
			slog.String("log_id", logID), slog.Any("error", err))
	}
}

// loadActiveHours fetches the active_hours_start, active_hours_end, and active_days
// columns for the given heartbeat config ID directly from the database.
// Returns (start=0, end=23, days=nil, false) if unavailable.
func (e *Engine) loadActiveHours(ctx context.Context, configID string) (start, end int, days []int, ok bool) {
	if e.dbPool == nil {
		return 0, 23, nil, false
	}
	row := e.dbPool.QueryRow(ctx,
		`SELECT active_hours_start, active_hours_end, active_days
		 FROM heartbeat_configs WHERE id=$1`,
		configID,
	)
	var pgStart, pgEnd int16
	var pgDays []int16
	if err := row.Scan(&pgStart, &pgEnd, &pgDays); err != nil {
		return 0, 23, nil, false
	}
	days = make([]int, len(pgDays))
	for i, d := range pgDays {
		days[i] = int(d)
	}
	return int(pgStart), int(pgEnd), days, true
}

// isWithinActiveHours returns true when the current time (in the engine timezone)
// falls within the configured active window for a heartbeat config.
// If active hours have not been configured, it always returns true.
func (e *Engine) isWithinActiveHours(ctx context.Context, cfg Config) bool {
	start, end, days, ok := e.loadActiveHours(ctx, cfg.ID)
	if !ok {
		return true
	}

	loc := e.timezone
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	hour := now.Hour()
	weekday := int(now.Weekday())

	// Check hour window.
	if hour < start || hour > end {
		return false
	}

	// Empty days list means all days are active.
	if len(days) == 0 {
		return true
	}
	for _, d := range days {
		if d == weekday {
			return true
		}
	}
	return false
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
