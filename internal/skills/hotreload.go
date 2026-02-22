package skills

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// HotReloader manages hot-reloading of skills from the filesystem.
type HotReloader struct {
	logger   *slog.Logger
	watchers map[string]*fsnotify.Watcher // botID -> watcher
	mu       sync.RWMutex
	onChange func(botID string) // callback when skills change
}

// NewHotReloader creates a new HotReloader instance.
func NewHotReloader(logger *slog.Logger, onChange func(botID string)) *HotReloader {
	if logger == nil {
		logger = slog.Default()
	}
	return &HotReloader{
		logger:   logger,
		watchers: make(map[string]*fsnotify.Watcher),
		onChange: onChange,
	}
}

// Watch starts watching a bot's skills directory for changes.
func (hr *HotReloader) Watch(ctx context.Context, botID string, skillsDir string) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	// Check if already watching
	if _, exists := hr.watchers[botID]; exists {
		return nil
	}

	// Create new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	// Add the skills directory
	if err := watcher.Add(skillsDir); err != nil {
		watcher.Close()
		return fmt.Errorf("watch directory: %w", err)
	}

	// Watch all subdirectories
	entries, err := os.ReadDir(skillsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				subDir := filepath.Join(skillsDir, entry.Name())
				_ = watcher.Add(subDir) // Best effort
			}
		}
	}

	hr.watchers[botID] = watcher

	// Start watching in background
	go hr.watchLoop(ctx, botID, watcher, skillsDir)

	hr.logger.Info("Started watching skills directory",
		slog.String("bot_id", botID),
		slog.String("dir", skillsDir))

	return nil
}

// watchLoop monitors file system events.
func (hr *HotReloader) watchLoop(ctx context.Context, botID string, watcher *fsnotify.Watcher, skillsDir string) {
	defer hr.stopWatching(botID)

	// Debounce rapid changes
	var debounceTimer *time.Timer
	debounceInterval := 500 * time.Millisecond

	triggerReload := func() {
		if hr.onChange != nil {
			hr.logger.Debug("Triggering skill reload",
				slog.String("bot_id", botID))
			hr.onChange(botID)
		}
	}

	for {
		select {
		case <-ctx.Done():
			hr.logger.Info("Stopping skill watcher",
				slog.String("bot_id", botID))
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only care about writes and creates
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			// Skip non-skill files
			ext := filepath.Ext(event.Name)
			if ext != ".md" && ext != ".json" {
				continue
			}

			hr.logger.Debug("Skill file changed",
				slog.String("bot_id", botID),
				slog.String("file", event.Name),
				slog.String("op", event.Op.String()))

			// If a new directory is created, watch it
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = watcher.Add(event.Name)
				}
			}

			// Debounce: reset timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceInterval, triggerReload)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			hr.logger.Warn("Watcher error",
				slog.String("bot_id", botID),
				slog.Any("error", err))
		}
	}
}

// stopWatching removes a bot's watcher.
func (hr *HotReloader) stopWatching(botID string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	if watcher, exists := hr.watchers[botID]; exists {
		watcher.Close()
		delete(hr.watchers, botID)
		hr.logger.Info("Stopped watching skills directory",
			slog.String("bot_id", botID))
	}
}

// Unwatch stops watching a bot's skills directory.
func (hr *HotReloader) Unwatch(botID string) {
	hr.stopWatching(botID)
}

// UnwatchAll stops all watchers.
func (hr *HotReloader) UnwatchAll() {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	for botID, watcher := range hr.watchers {
		watcher.Close()
		hr.logger.Info("Stopped watching skills directory",
			slog.String("bot_id", botID))
	}
	hr.watchers = make(map[string]*fsnotify.Watcher)
}

// IsWatching returns true if the bot's skills directory is being watched.
func (hr *HotReloader) IsWatching(botID string) bool {
	hr.mu.RLock()
	defer hr.mu.RUnlock()
	_, exists := hr.watchers[botID]
	return exists
}
