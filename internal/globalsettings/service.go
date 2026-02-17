package globalsettings

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
)

const KeyTimezone = "timezone"

// Service manages global key-value settings backed by PostgreSQL.
// It caches the timezone value in memory and provides a callback
// mechanism so other components (e.g. Resolver) can react to changes.
type Service struct {
	queries     *sqlc.Queries
	cfg         config.Config
	logger      *slog.Logger

	mu          sync.RWMutex
	timezone    string
	timezoneLoc *time.Location

	onTimezoneChange []func(tz string, loc *time.Location)
}

// NewService creates a new global settings service.
func NewService(log *slog.Logger, queries *sqlc.Queries, cfg config.Config) *Service {
	return &Service{
		queries: queries,
		cfg:     cfg,
		logger:  log.With(slog.String("service", "global_settings")),
	}
}

// Init loads the timezone from the database. If not set, falls back to
// config.toml, then to UTC. Should be called once at startup.
func (s *Service) Init(ctx context.Context) error {
	row, err := s.queries.GetGlobalSetting(ctx, KeyTimezone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	tzName := ""
	if err == nil && row.Value != "" {
		tzName = row.Value
	} else {
		tzName = s.cfg.Server.Timezone
	}
	if tzName == "" {
		tzName = "UTC"
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		s.logger.Warn("invalid timezone in DB/config, falling back to UTC",
			slog.String("timezone", tzName), slog.Any("error", err))
		tzName = "UTC"
		loc = time.UTC
	}

	s.mu.Lock()
	s.timezone = tzName
	s.timezoneLoc = loc
	s.mu.Unlock()

	s.logger.Info("timezone initialized", slog.String("timezone", tzName))
	return nil
}

// GetTimezone returns the current timezone name and location.
func (s *Service) GetTimezone() (string, *time.Location) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.timezone, s.timezoneLoc
}

// SetTimezone updates the timezone in the database and memory.
// It fires registered callbacks so other components can react.
func (s *Service) SetTimezone(ctx context.Context, tzName string) error {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return err
	}

	_, err = s.queries.UpsertGlobalSetting(ctx, sqlc.UpsertGlobalSettingParams{
		Key:   KeyTimezone,
		Value: tzName,
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.timezone = tzName
	s.timezoneLoc = loc
	s.mu.Unlock()

	s.logger.Info("timezone updated", slog.String("timezone", tzName))

	for _, fn := range s.onTimezoneChange {
		fn(tzName, loc)
	}

	return nil
}

// OnTimezoneChange registers a callback that fires when timezone is updated.
func (s *Service) OnTimezoneChange(fn func(tz string, loc *time.Location)) {
	s.onTimezoneChange = append(s.onTimezoneChange, fn)
}
