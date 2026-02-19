package message

import (
	"strings"
	"sync"
	"time"
)

const (
	// DefaultGroupDebounceWindow is the default time window for merging group
	// messages from the same conversation before dispatching them to the agent.
	DefaultGroupDebounceWindow = 300 * time.Millisecond

	// groupMessageSeparator is inserted between merged messages in the same window.
	groupMessageSeparator = "\n---\n"
)

// PendingGroup holds buffered messages and the single timer that will fire them.
type PendingGroup struct {
	mu      sync.Mutex
	texts   []string
	timer   *time.Timer
	execute func(mergedText string)
}

// Append adds a message text to the buffer and resets the timer.
// Returns true if the timer was reset (caller should NOT trigger processing itself).
func (pg *PendingGroup) Append(text string, window time.Duration, execute func(mergedText string)) bool {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.texts = append(pg.texts, text)
	pg.execute = execute
	if pg.timer != nil {
		pg.timer.Reset(window)
		return true
	}
	// First message in window: start timer.
	pg.timer = time.AfterFunc(window, func() {
		pg.mu.Lock()
		merged := strings.Join(pg.texts, groupMessageSeparator)
		fn := pg.execute
		pg.mu.Unlock()
		if fn != nil {
			fn(merged)
		}
	})
	// First message: the caller should also NOT process it directly; the timer handles it.
	return true
}

// GroupDebouncer batches group messages by a chatID key, merging rapid bursts
// into a single agent invocation to reduce redundant processing.
//
// Direct messages (DM) bypass this entirely — they are always dispatched
// immediately by the caller.
type GroupDebouncer struct {
	mu      sync.Mutex
	window  time.Duration
	pending map[string]*PendingGroup // key: chatID
}

// NewGroupDebouncer creates a debouncer with the given window duration.
// Pass 0 or a negative duration to use the DefaultGroupDebounceWindow.
func NewGroupDebouncer(window time.Duration) *GroupDebouncer {
	if window <= 0 {
		window = DefaultGroupDebounceWindow
	}
	return &GroupDebouncer{
		window:  window,
		pending: make(map[string]*PendingGroup),
	}
}

// Submit buffers a group message for the given key (typically chatID) and
// schedules dispatch after the debounce window. The `execute` callback will
// be called with the merged text after the window expires.
//
// Returns true in all cases — the caller should return immediately and let the
// debouncer drive execution.
func (d *GroupDebouncer) Submit(key, text string, execute func(mergedText string)) {
	d.mu.Lock()
	pg, ok := d.pending[key]
	if !ok {
		pg = &PendingGroup{}
		d.pending[key] = pg
		// Register cleanup: remove from map after execution.
		originalExecute := execute
		execute = func(merged string) {
			d.mu.Lock()
			delete(d.pending, key)
			d.mu.Unlock()
			originalExecute(merged)
		}
	}
	d.mu.Unlock()

	pg.Append(text, d.window, execute)
}

// Flush cancels any pending timer for the given key and discards buffered messages.
// Useful for testing or when a conversation is deleted.
func (d *GroupDebouncer) Flush(key string) {
	d.mu.Lock()
	pg, ok := d.pending[key]
	if ok {
		delete(d.pending, key)
	}
	d.mu.Unlock()
	if ok {
		pg.mu.Lock()
		if pg.timer != nil {
			pg.timer.Stop()
		}
		pg.mu.Unlock()
	}
}
