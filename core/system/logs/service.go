package logs

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/logs/model"
	"github.com/asaidimu/hestia/core/system/policies"
)

// LogsService provides log querying and streaming.
type LogsService struct {
	reader *Reader
	ring   *RingBuffer
	logger *zap.Logger
}

// NewLogsService creates a LogsService. ring may be nil if streaming is not needed.
func NewLogsService(logPath string, ring *RingBuffer, logger *zap.Logger) *LogsService {
	return &LogsService{
		reader: NewReader(logPath),
		ring:   ring,
		logger: logger,
	}
}

// Query searches historical logs from the NDJSON file.
//
// @hestia.register(
//
//	name="system:logs:list",
//	intent="query",
//	rule="administrator",
//	description="Query application logs",
//	output="model.LogListOutput",
//
// )
func (s *LogsService) Query(ctx context.Context, msg abstract.Message, input *model.LogQueryInput) (*abstract.Result, error) {
	q := Query{
		Level:  input.Level,
		Search: input.Search,
		Limit:  input.Limit,
		Offset: input.Offset,
	}

	if input.From != "" {
		if t, err := time.Parse(time.RFC3339, input.From); err == nil {
			q.From = t
		}
	}
	if input.To != "" {
		if t, err := time.Parse(time.RFC3339, input.To); err == nil {
			q.To = t
		}
	}

	result, err := s.reader.Query(q)
	if err != nil {
		return nil, err
	}

	views := make([]model.LogEntryView, len(result.Entries))
	for i, e := range result.Entries {
		views[i] = model.LogEntryView{
			Level:  e.Level,
			TS:     e.TS,
			Caller: e.Caller,
			Msg:    e.Msg,
			Fields: e.Fields,
			Extra:  e.Extra,
		}
	}

	doc := model.LogListDocument{
		Entries: views,
		Total:   result.Total,
		HasMore: result.HasMore,
	}
	return dispatch.NewDocumentResultFrom(&doc)
}

// Stream streams live log entries from the in-memory ring buffer.
//
// @hestia.register(
//
//	name="system:logs:stream",
//	intent="stream",
//	rule="administrator",
//	description="Stream live log entries",
//	input="model.LogStreamInput",
//	output="model.LogStreamOutput",
//
// )
func (s *LogsService) Stream(ctx context.Context, msg abstract.Message, input *model.LogStreamInput) (*abstract.Result, error) {
	docCh := make(chan *document.Document, 64)

	sendEntry := func(e LogEntry) {
		if input.Level != "" && e.Level != input.Level {
			return
		}
		view := model.LogEntryView{
			Level:  e.Level,
			TS:     e.TS,
			Caller: e.Caller,
			Msg:    e.Msg,
			Fields: e.Fields,
			Extra:  e.Extra,
		}
		result, err := dispatch.NewDocumentResultFrom(&view)
		if err != nil {
			return
		}
		select {
		case docCh <- result.Document:
		default:
		}
	}

	// Send recent entries first
	for _, e := range s.ring.Recent(50) {
		sendEntry(e)
	}

	// Poll ring buffer for new entries
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		lastLen := s.ring.Len()

		for {
			select {
			case <-ctx.Done():
				close(docCh)
				return
			case <-ticker.C:
				all := s.ring.Recent(0)
				if len(all) > lastLen {
					diff := len(all) - lastLen
					if diff > len(all) {
						diff = len(all)
					}
					for _, e := range all[len(all)-diff:] {
						sendEntry(e)
					}
					lastLen = len(all)
				}
			}
		}
	}()

	return &abstract.Result{DocumentChannel: docCh}, nil
}

// Registrations returns the message registrations for the logs feature.
func (s *LogsService) Registrations() []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{
			Name:        "system:logs:list",
			Description: "Query application logs",
			Intent:      abstract.Query,
			Enabled:     true,
			Input: abstract.Input{
				Schema: dispatch.SchemaFromTypeWithTag[model.LogQueryInput]("input"),
			},
			Output:  dispatch.SchemaFromType[model.LogListOutput](),
			Handler: dispatch.Handle[model.LogQueryInput](s.Query),
		},
		{
			Name:        "system:logs:stream",
			Description: "Stream live log entries",
			Intent:      abstract.Stream,
			Enabled:     true,
			Input: abstract.Input{
				Schema: dispatch.SchemaFromTypeWithTag[model.LogStreamInput]("input"),
			},
			Output:  dispatch.SchemaFromType[model.LogStreamOutput](),
			Handler: dispatch.Handle[model.LogStreamInput](s.Stream),
		},
	}
}

// LogPolicyBindings returns policy bindings for the logs feature.
func LogPolicyBindings() []policies.Binding {
	return Policies()
}
