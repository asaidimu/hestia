package logs

import (
	"sync"
	"time"
)

// LogEntry is a single parsed log line.
type LogEntry struct {
	Level  string         `json:"level"`
	TS     float64        `json:"ts"`
	Caller string         `json:"caller"`
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
}

// Time returns the entry's timestamp as time.Time.
func (e *LogEntry) Time() time.Time {
	return time.Unix(0, int64(e.TS*1e9))
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
