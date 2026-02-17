package automation

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// CronPool wraps robfig/cron as a shared, process-wide scheduler.
// Both schedule.Service and heartbeat.Engine use CronPool to register jobs,
// eliminating the need for separate timer/cron management in each subsystem.
type CronPool struct {
	cron   *cron.Cron
	parser cron.Parser
	logger *slog.Logger

	mu    sync.Mutex
	jobs  map[string]cron.EntryID
	locks map[string]*sync.Mutex
}

// NewCronPool creates an idle CronPool. Call Start() to begin scheduling.
// The loc parameter sets the timezone for cron pattern interpretation.
// If loc is nil, time.UTC is used.
func NewCronPool(log *slog.Logger, loc *time.Location) *CronPool {
	if loc == nil {
		loc = time.UTC
	}
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour |
			cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	c := cron.New(cron.WithParser(parser), cron.WithLocation(loc))
	return &CronPool{
		cron:   c,
		parser: parser,
		logger: log.With(slog.String("component", "cron_pool"), slog.String("timezone", loc.String())),
		jobs:   make(map[string]cron.EntryID),
		locks:  make(map[string]*sync.Mutex),
	}
}

// ValidatePattern checks if a cron pattern is syntactically valid.
func (p *CronPool) ValidatePattern(pattern string) error {
	_, err := p.parser.Parse(pattern)
	return err
}

// Add registers a job under the given id with the given cron pattern.
// The callback fn is wrapped with a per-job mutex so that overlapping
// triggers are skipped (not queued), preventing concurrent execution
// of the same job when a previous run hasn't finished.
func (p *CronPool) Add(id, pattern string, fn func()) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If there's already a job with this id, remove it first.
	if entryID, ok := p.jobs[id]; ok {
		p.cron.Remove(entryID)
		delete(p.jobs, id)
	}

	jobMu := &sync.Mutex{}
	p.locks[id] = jobMu

	wrapped := func() {
		if !jobMu.TryLock() {
			p.logger.Debug("skipping overlapping trigger",
				slog.String("job_id", id),
			)
			return
		}
		defer jobMu.Unlock()
		fn()
	}

	entryID, err := p.cron.AddFunc(pattern, wrapped)
	if err != nil {
		delete(p.locks, id)
		return err
	}
	p.jobs[id] = entryID
	return nil
}

// Remove unregisters a job by id. Safe to call with unknown ids.
func (p *CronPool) Remove(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if entryID, ok := p.jobs[id]; ok {
		p.cron.Remove(entryID)
		delete(p.jobs, id)
		delete(p.locks, id)
	}
}

// Replace is a convenience method: Remove + Add. If the new pattern is
// invalid, the old job is already removed (caller should handle the error).
func (p *CronPool) Replace(id, pattern string, fn func()) error {
	p.Remove(id)
	return p.Add(id, pattern, fn)
}

// Has returns true if a job with the given id is registered.
func (p *CronPool) Has(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.jobs[id]
	return ok
}

// Start begins the cron scheduler. Idempotent if already started.
func (p *CronPool) Start() {
	p.cron.Start()
}

// Stop halts the cron scheduler, waiting for running jobs to complete.
// Returns a context that is done when all jobs have finished.
func (p *CronPool) Stop() context.Context {
	return p.cron.Stop()
}
