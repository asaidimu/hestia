package workflows

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

// WorkflowCompiledView is the result of compiling a workflow graph.
type WorkflowCompiledView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	WorkflowID             string `anansi:"workflow_id"`
	Label                  string `anansi:"label"`
	Triggers               int    `anansi:"triggers"`
	Pipelines              int    `anansi:"pipelines"`
}

// WorkflowRegisteredView is the result of registering a workflow definition.
type WorkflowRegisteredView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string `anansi:"id"`
	WorkflowID             string `anansi:"workflow_id"`
	Message                string `anansi:"message"`
}

// WorkflowRunStartedView is the result of starting an ad-hoc run.
type WorkflowRunStartedView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	RunID                  string `anansi:"run_id"`
}

// WorkflowEventEmittedView is the result of emitting a trigger event.
type WorkflowEventEmittedView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	OK                     bool `anansi:"ok"`
}

// WorkflowAbortedView is the result of aborting a run.
type WorkflowAbortedView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	OK                     bool `anansi:"ok"`
}

// WorkflowRunMetadataView is the metadata for a single run.
type WorkflowRunMetadataView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	RunID                  string  `anansi:"run_id"`
	PipelineID             string  `anansi:"pipeline_id"`
	StartTime              int64   `anansi:"start_time"`
	EndTime                *int64  `anansi:"end_time,omitempty"`
	EventCount             int64   `anansi:"event_count"`
	Status                 string  `anansi:"status"`
	Metadata               map[string]any `anansi:"metadata,omitempty"`
}

// WorkflowRunOutcomeView is the settlement status of a run.
type WorkflowRunOutcomeView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	OK                     bool           `anansi:"ok"`
	RunID                  string         `anansi:"run_id"`
	Status                 string         `anansi:"status"`
	FinalState             map[string]any `anansi:"final_state,omitempty"`
	Error                  string         `anansi:"error,omitempty"`
}

// WorkflowRunEventsView contains timeline events for a run.
type WorkflowRunEventsView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Events                 []map[string]any `anansi:"events"`
}

// WorkflowRunStoreView contains the live state of a run.
type WorkflowRunStoreView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	State                  map[string]any `anansi:"state"`
}

// WorkflowDefinitionView is a stored workflow definition.
type WorkflowDefinitionView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string         `anansi:"_id"`
	Name                   string         `anansi:"name"`
	Description            *string        `anansi:"description,omitempty"`
	Nodes                  map[string]any `anansi:"nodes"`
	Edges                  map[string]any `anansi:"edges"`
	TenantID               *string        `anansi:"tenant_id,omitempty"`
	CreatedAt              *int64         `anansi:"created_at,omitempty"`
	UpdatedAt              *int64         `anansi:"updated_at,omitempty"`
}

// WorkflowMessageOutput is a generic message response.
type WorkflowMessageOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}
