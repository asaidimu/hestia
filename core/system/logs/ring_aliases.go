package logs

import (
	"go.uber.org/zap/zapcore"

	"github.com/asaidimu/hestia/core/runtime/logsink"
)

// The ring buffer and sink are boot-level infrastructure and moved to
// core/runtime/logsink (audit A-3). Type aliases keep this feature
// package's service, reader, and sanitization code (and its generated
// registrations) compiling against the same identities — an alias is the
// same type, not a wrapper, so *logs.RingBuffer and *logsink.RingBuffer
// are interchangeable everywhere, including DI resolution.
type (
	LogEntry   = logsink.LogEntry
	RingBuffer = logsink.RingBuffer
	Sink       = logsink.Sink
)

// NewRingBuffer creates the boot log ring buffer (delegates to logsink).
func NewRingBuffer(cap int) *RingBuffer { return logsink.NewRingBuffer(cap) }

// NewSink creates a sink that buffers entries and optionally writes to a
// downstream zapcore.WriteSyncer (delegates to logsink).
func NewSink(ring *RingBuffer, downstream zapcore.WriteSyncer) *Sink {
	return logsink.NewSink(ring, downstream)
}
