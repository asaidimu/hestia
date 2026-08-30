// Package logsink holds the boot-seeded in-memory log ring buffer and
// its zapcore sink (audit A-3). This is infrastructure the lowest layer
// (core/internal/boot) constructs to capture server logs; it used to live
// in core/system/logs — feature-land — which forced the boot layer to
// import a feature package.
package logsink

import (
	"encoding/json"
	"sync"
	"time"
)

// LogEntry is a single parsed log line.
// Known fields are extracted explicitly; everything else lands in Extra.
type LogEntry struct {
	Level  string         `json:"level"`
	TS     float64        `json:"ts"`
	Caller string         `json:"caller"`
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
	Extra  map[string]any `json:"-"` // all other top-level JSON keys
}

// Time returns the entry's timestamp as time.Time.
func (e *LogEntry) Time() time.Time {
	return time.Unix(0, int64(e.TS*1e9))
}

// UnmarshalJSON implements json.Unmarshaler so that every top-level key
// that isn't level/ts/caller/msg/fields ends up in Extra.
func (e *LogEntry) UnmarshalJSON(data []byte) error {
	// Decode into a flat map first.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if v, ok := raw["level"].(string); ok {
		e.Level = v
	}
	if v, ok := raw["ts"].(float64); ok {
		e.TS = v
	}
	if v, ok := raw["caller"].(string); ok {
		e.Caller = v
	}
	if v, ok := raw["msg"].(string); ok {
		e.Msg = v
	}
	if v, ok := raw["fields"].(map[string]any); ok {
		e.Fields = v
	}

	// Everything else goes into Extra.
	known := map[string]bool{"level": true, "ts": true, "caller": true, "msg": true, "fields": true}
	e.Extra = make(map[string]any, len(raw)-len(known))
	for k, v := range raw {
		if !known[k] {
			e.Extra[k] = v
		}
	}
	return nil
}

// RingBuffer is a fixed-capacity concurrent ring buffer of log entries.
type RingBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	head    int
	size    int
	cap     int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(cap int) *RingBuffer {
	return &RingBuffer{
		entries: make([]LogEntry, cap),
		cap:     cap,
	}
}

// Add appends an entry to the ring buffer.
func (r *RingBuffer) Add(entry LogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[r.head] = entry
	r.head = (r.head + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
}

// Recent returns the last n entries in chronological order.
// If n <= 0 or n > r.size, all entries are returned.
func (r *RingBuffer) Recent(n int) []LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n <= 0 || n > r.size {
		n = r.size
	}

	result := make([]LogEntry, n)
	start := (r.head - n + r.cap) % r.cap
	for i := 0; i < n; i++ {
		result[i] = r.entries[(start+i)%r.cap]
	}
	return result
}

// Len returns the number of entries currently in the buffer.
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}
