package logs

import (
	"encoding/json"
	"sync"

	"go.uber.org/zap/zapcore"
)

// Sink is a zapcore.WriteSyncer that stores entries in a RingBuffer
// and fans out to an optional downstream WriteSyncer.
type Sink struct {
	ring       *RingBuffer
	downstream zapcore.WriteSyncer
	mu         sync.Mutex
}

// NewSink creates a sink that buffers entries and optionally writes to downstream.
func NewSink(ring *RingBuffer, downstream zapcore.WriteSyncer) *Sink {
	return &Sink{ring: ring, downstream: downstream}
}

// Write implements io.Writer. It parses the JSON line and stores it in the ring.
func (s *Sink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var entry LogEntry
	if err := json.Unmarshal(p, &entry); err == nil {
		s.ring.Add(entry)
	}

	if s.downstream != nil {
		return s.downstream.Write(p)
	}
	return len(p), nil
}

// Sync implements zapcore.WriteSyncer.
func (s *Sink) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.downstream != nil {
		return s.downstream.Sync()
	}
	return nil
}
