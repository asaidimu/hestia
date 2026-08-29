package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/workflows/model"

	"github.com/asaidimu/hermes/pkg/compiler"
	"github.com/asaidimu/hermes/pkg/events"
	"github.com/asaidimu/hermes/pkg/nodekit"
	hermesruntime "github.com/asaidimu/hermes/pkg/runtime"
	"github.com/asaidimu/hermes/pkg/timeline"
	_ "github.com/asaidimu/hermes/pkg/nodes" // register built-in node types
)

// WorkflowsService exposes the hermes workflow engine as hestia message handlers.
type WorkflowsService struct {
	defs    *model.SystemWorkflowDefinitions
	runtime *hermesruntime.WorkflowRuntime
}

// NewWorkflowsService constructs the service, resolving dependencies from DI.
func NewWorkflowsService(rt abstract.Container) (*WorkflowsService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	defs, err := model.InitSystemWorkflowDefinitionsModel(persist, nil)
	if err != nil {
		return nil, err
	}
	runtime := abstract.MustResolve[*hermesruntime.WorkflowRuntime](rt)
	return &WorkflowsService{defs: defs, runtime: runtime}, nil
}

// NewWorkflowsServiceForTest constructs the service with explicit dependencies for testing.
func NewWorkflowsServiceForTest(defs *model.SystemWorkflowDefinitions, rt *hermesruntime.WorkflowRuntime) *WorkflowsService {
	return &WorkflowsService{defs: defs, runtime: rt}
}

// Compile compiles a workflow graph into a pipeline definition.
//
// @hestia.register(
//   name="system:workflows:definition:compile",
//   intent="create",
//   rule="administrator",
//   description="Compile a workflow graph into a pipeline definition",
// )
func (s *WorkflowsService) Compile(ctx context.Context, msg abstract.Message, input *WorkflowCompileInput) (*WorkflowCompiledView, error) {
	nodes, err := toCompilerNodes(input.Nodes)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_INVALID_NODES", fmt.Sprintf("invalid nodes: %v", err))
	}
	edges, err := toCompilerEdges(input.Edges)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_INVALID_EDGES", fmt.Sprintf("invalid edges: %v", err))
	}

	wf, err := compiler.Compile(nodes, edges, nil)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_COMPILE_FAILED", fmt.Sprintf("compile failed: %v", err))
	}

	return document.New(&WorkflowCompiledView{
		WorkflowID: wf.ID,
		Label:      wf.Label,
		Triggers:   len(wf.Triggers),
		Pipelines:  len(wf.Pipelines),
	}), nil
}

// Register stores a workflow definition and registers it with the runtime.
//
// @hestia.register(
//   name="system:workflows:definition:create",
//   intent="create",
//   rule="administrator",
//   description="Store and register a workflow definition",
// )
func (s *WorkflowsService) Register(ctx context.Context, msg abstract.Message, input *WorkflowRegisterInput) (*WorkflowRegisteredView, error) {
	nodes, err := toCompilerNodes(input.Nodes)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_INVALID_NODES", fmt.Sprintf("invalid nodes: %v", err))
	}
	edges, err := toCompilerEdges(input.Edges)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_INVALID_EDGES", fmt.Sprintf("invalid edges: %v", err))
	}

	wf, err := compiler.Compile(nodes, edges, nil)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_COMPILE_FAILED", fmt.Sprintf("compile failed: %v", err))
	}

	if err := s.runtime.Register(wf, hermesruntime.RegisterOptions{
		Mode: hermesruntime.Mode{Type: "serialized"},
	}); err != nil {
		return nil, common.NewSystemError("WORKFLOW_REGISTER_FAILED", fmt.Sprintf("register failed: %v", err))
	}

	doc := data.MustNewDocument(map[string]any{
		"name":        input.Name,
		"description": input.Description,
		"nodes":       input.Nodes,
		"edges":       input.Edges,
	})
	saved, err := s.defs.CreateDefinition(ctx, doc)
	if err != nil {
		return nil, err
	}

	return document.New(&WorkflowRegisteredView{
		ID:         saved.ID(),
		WorkflowID: wf.ID,
		Message:    "workflow registered",
	}), nil
}

// Deregister removes a workflow from the runtime.
//
// @hestia.register(
//   name="system:workflows:definition:delete",
//   intent="delete",
//   rule="administrator",
//   description="Deregister a workflow definition",
//   resource_id="id",
// )
func (s *WorkflowsService) Deregister(ctx context.Context, msg abstract.Message, input *WorkflowDefinitionDeleteInput) (*WorkflowMessageOutput, error) {
	if input.ID == "" {
		return nil, common.NewSystemError("WORKFLOW_ID_REQUIRED", "id is required")
	}

	existing, err := s.defs.GetDefinition(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, common.NewSystemError("WORKFLOW_NOT_FOUND", fmt.Sprintf("workflow definition %q not found", input.ID))
	}

	s.runtime.Deregister(input.ID)

	if err := s.defs.DeleteDefinition(ctx, input.ID); err != nil {
		return nil, err
	}

	return document.New(&WorkflowMessageOutput{Message: "workflow deregistered"}), nil
}

// ListDefinitions lists all stored workflow definitions.
//
// @hestia.register(
//   name="system:workflows:definition:list",
//   intent="read",
//   rule="administrator",
//   description="List all workflow definitions",
// )
func (s *WorkflowsService) ListDefinitions(ctx context.Context, msg abstract.Message, input *WorkflowDefinitionListInput) ([]*document.Document, error) {
	docs, err := s.defs.ListDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// GetDefinition returns a single workflow definition.
//
// @hestia.register(
//   name="system:workflows:definition:get",
//   intent="read",
//   rule="administrator",
//   description="Get a workflow definition by ID",
//   resource_id="id",
// )
func (s *WorkflowsService) GetDefinition(ctx context.Context, msg abstract.Message, input *WorkflowDefinitionGetInput) (*WorkflowDefinitionView, error) {
	if input.ID == "" {
		return nil, common.NewSystemError("WORKFLOW_ID_REQUIRED", "id is required")
	}

	doc, err := s.defs.GetDefinition(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, common.NewSystemError("WORKFLOW_NOT_FOUND", fmt.Sprintf("workflow definition %q not found", input.ID))
	}

	var view WorkflowDefinitionView
	if err := doc.BindTo(&view); err != nil {
		return nil, err
	}
	view.ID = doc.ID()
	return document.New(&view), nil
}

// Run compiles and executes a workflow graph ad-hoc.
//
// @hestia.register(
//   name="system:workflows:runtime:run",
//   intent="create",
//   rule="administrator",
//   description="Compile and run a workflow graph",
// )
func (s *WorkflowsService) Run(ctx context.Context, msg abstract.Message, input *WorkflowRunInput) (*WorkflowRunStartedView, error) {
	nodes, err := toCompilerNodes(input.Nodes)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_INVALID_NODES", fmt.Sprintf("invalid nodes: %v", err))
	}
	edges, err := toCompilerEdges(input.Edges)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_INVALID_EDGES", fmt.Sprintf("invalid edges: %v", err))
	}

	result, err := s.runtime.Run(ctx, nodes, edges)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_RUN_FAILED", fmt.Sprintf("run failed: %v", err))
	}

	return document.New(&WorkflowRunStartedView{
		RunID: result.RunID,
	}), nil
}

// EmitEvent dispatches a trigger event to the workflow runtime bus.
//
// @hestia.register(
//   name="system:workflows:runtime:events",
//   intent="create",
//   rule="administrator",
//   description="Emit a trigger event to the workflow runtime",
// )
func (s *WorkflowsService) EmitEvent(ctx context.Context, msg abstract.Message, input *WorkflowEventInput) (*WorkflowEventEmittedView, error) {
	if input.Type == "" {
		return nil, common.NewSystemError("WORKFLOW_EVENT_TYPE_REQUIRED", "event type is required")
	}

	s.runtime.Bus().Emit(ctx, input.Type, events.PipelineEvent{
		Payload: input.Payload,
	})

	return document.New(&WorkflowEventEmittedView{OK: true}), nil
}

// Abort stops a running pipeline.
//
// @hestia.register(
//   name="system:workflows:runtime:abort",
//   intent="delete",
//   rule="administrator",
//   description="Abort a running workflow",
//   resource_id="run_id",
// )
func (s *WorkflowsService) Abort(ctx context.Context, msg abstract.Message, input *WorkflowAbortInput) (*WorkflowAbortedView, error) {
	if input.RunID == "" {
		return nil, common.NewSystemError("WORKFLOW_RUN_ID_REQUIRED", "run_id is required")
	}

	s.runtime.AbortRun(input.RunID)

	return document.New(&WorkflowAbortedView{OK: true}), nil
}

// ListRuns lists all execution runs.
//
// @hestia.register(
//   name="system:workflows:run:list",
//   intent="read",
//   rule="administrator",
//   description="List all workflow runs",
// )
func (s *WorkflowsService) ListRuns(ctx context.Context, msg abstract.Message, input *WorkflowRunListInput) ([]*document.Document, error) {
	metas, err := s.runtime.ListRuns(ctx)
	if err != nil {
		return nil, err
	}

	docs := make([]*document.Document, 0, len(metas))
	for _, m := range metas {
		view := map[string]any{
			"run_id":      m.RunID,
			"pipeline_id": m.PipelineID,
			"start_time":  m.StartTime,
			"end_time":    m.EndTime,
			"event_count": m.EventCount,
			"status":      string(m.Status),
			"metadata":    m.Metadata,
		}
		docs = append(docs, document.NewRecordView(view))
	}
	return docs, nil
}

// GetRun returns metadata for a single run.
//
// @hestia.register(
//   name="system:workflows:run:get",
//   intent="read",
//   rule="administrator",
//   description="Get workflow run metadata",
//   resource_id="run_id",
// )
func (s *WorkflowsService) GetRun(ctx context.Context, msg abstract.Message, input *WorkflowRunGetInput) (*WorkflowRunMetadataView, error) {
	if input.RunID == "" {
		return nil, common.NewSystemError("WORKFLOW_RUN_ID_REQUIRED", "run_id is required")
	}

	meta, err := s.runtime.GetRunMeta(ctx, input.RunID)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_RUN_NOT_FOUND", fmt.Sprintf("run %q not found", input.RunID))
	}

	return document.New(&WorkflowRunMetadataView{
		RunID:      meta.RunID,
		PipelineID: meta.PipelineID,
		StartTime:  meta.StartTime,
		EndTime:    meta.EndTime,
		EventCount: meta.EventCount,
		Status:     string(meta.Status),
		Metadata:   meta.Metadata,
	}), nil
}

// GetOutcome returns the settlement status of a run.
//
// @hestia.register(
//   name="system:workflows:run:outcome",
//   intent="read",
//   rule="administrator",
//   description="Get workflow run outcome",
//   resource_id="run_id",
// )
func (s *WorkflowsService) GetOutcome(ctx context.Context, msg abstract.Message, input *WorkflowRunOutcomeInput) (*WorkflowRunOutcomeView, error) {
	if input.RunID == "" {
		return nil, common.NewSystemError("WORKFLOW_RUN_ID_REQUIRED", "run_id is required")
	}

	outcome, ok := s.runtime.GetRunOutcome(input.RunID)
	if !ok {
		return nil, common.NewSystemError("WORKFLOW_RUN_NOT_FOUND", fmt.Sprintf("run %q not found or not completed", input.RunID))
	}

	var errStr string
	if outcome.Error != nil {
		errStr = outcome.Error.Error()
	}

	return document.New(&WorkflowRunOutcomeView{
		OK:         outcome.OK,
		RunID:      outcome.RunID,
		Status:     outcome.Status,
		FinalState: outcome.FinalState,
		Error:      errStr,
	}), nil
}

// GetRunEvents returns timeline events for a run.
//
// @hestia.register(
//   name="system:workflows:run:events",
//   intent="read",
//   rule="administrator",
//   description="Get timeline events for a workflow run",
//   resource_id="run_id",
// )
func (s *WorkflowsService) GetRunEvents(ctx context.Context, msg abstract.Message, input *WorkflowRunEventsInput) (*WorkflowRunEventsView, error) {
	if input.RunID == "" {
		return nil, common.NewSystemError("WORKFLOW_RUN_ID_REQUIRED", "run_id is required")
	}

	timelineEvents, err := s.runtime.GetEvents(ctx, input.RunID, 0, 10000)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_RUN_EVENTS_FAILED", fmt.Sprintf("failed to get events: %v", err))
	}

	eventsOut := make([]map[string]any, 0, len(timelineEvents))
	for _, e := range timelineEvents {
		eventsOut = append(eventsOut, map[string]any{
			"run_id":    e.RunID,
			"seq":       e.Seq,
			"timestamp": e.Timestamp,
			"source":    string(e.Source),
			"type":      e.Type,
			"payload":   e.Payload,
			"delta":     e.Delta,
		})
	}

	return document.New(&WorkflowRunEventsView{Events: eventsOut}), nil
}

// GetRunStore returns the live state of a run.
//
// @hestia.register(
//   name="system:workflows:run:store",
//   intent="read",
//   rule="administrator",
//   description="Get the live state of a workflow run",
//   resource_id="run_id",
// )
func (s *WorkflowsService) GetRunStore(ctx context.Context, msg abstract.Message, input *WorkflowRunStoreInput) (*WorkflowRunStoreView, error) {
	if input.RunID == "" {
		return nil, common.NewSystemError("WORKFLOW_RUN_ID_REQUIRED", "run_id is required")
	}

	store := s.runtime.Store(input.RunID)
	if store == nil {
		return nil, common.NewSystemError("WORKFLOW_RUN_NOT_FOUND", fmt.Sprintf("run %q not found", input.RunID))
	}

	state := make(map[string]any)
	// Read all keys from the store
	_ = store.Update(ctx, func(m map[string]any) error {
		for k, v := range m {
			state[k] = v
		}
		return nil
	})

	return document.New(&WorkflowRunStoreView{State: state}), nil
}

// ---------------------------------------------------------------------------
// SSE streaming
// ---------------------------------------------------------------------------

// Stream streams timeline events for a run in real-time via SSE.
//
// It first replays all existing events, then subscribes to the runtime event
// bus and forwards new events as they arrive. The stream closes automatically
// on terminal events (pipeline:success, pipeline:failure, pipeline:pause) or
// when the client disconnects.
//
// @hestia.register(
//   name="system:workflows:run:stream",
//   intent="stream",
//   rule="administrator",
//   description="Stream workflow run events in real-time via SSE",
//   resource_id="run_id",
// )
func (s *WorkflowsService) Stream(ctx context.Context, msg abstract.Message, input *WorkflowRunStreamInput) (*abstract.Result, error) {
	if input.RunID == "" {
		return nil, common.NewSystemError("WORKFLOW_RUN_ID_REQUIRED", "run_id is required")
	}

	// Verify the run exists
	meta, err := s.runtime.GetRunMeta(ctx, input.RunID)
	if err != nil {
		return nil, common.NewSystemError("WORKFLOW_RUN_NOT_FOUND", fmt.Sprintf("run %q not found", input.RunID))
	}

	eventCh := make(chan *document.Document, 64)

	go func() {
		defer close(eventCh)

		// Phase 1: Replay existing events
		existingEvents, err := s.runtime.GetEvents(ctx, input.RunID, 0, 0)
		if err == nil {
			for _, e := range existingEvents {
				doc := document.NewRecordView(timelineEventToMap(e), ctx)
				select {
				case eventCh <- doc:
				case <-ctx.Done():
					return
				}
			}
		}

		// If run is already terminal, no need to subscribe to live events
		if meta.Status != "recording" {
			return
		}

		// Phase 2: Subscribe to live bus events
		done := make(chan struct{})
		unsub := s.runtime.Bus().Subscribe("*", func(_ context.Context, evt events.PipelineEvent) error {
			if evt.RunID != input.RunID {
				return nil
			}
			m := map[string]any{
				"runId":      evt.RunID,
				"type":       evt.Type,
				"timestamp":  evt.Timestamp,
				"pipelineId": evt.PipelineID,
				"path":       evt.Path,
			}
			if evt.Payload != nil {
				m["payload"] = evt.Payload
			}
			if evt.Duration > 0 {
				m["duration"] = evt.Duration
			}
			doc := document.NewRecordView(m, ctx)
			select {
			case eventCh <- doc:
			case <-ctx.Done():
				return nil
			}

			// Close on terminal events
			if evt.Type == "pipeline:success" || evt.Type == "pipeline:failure" || evt.Type == "pipeline:pause" {
				select {
				case <-done:
				default:
					close(done)
				}
			}
			return nil
		})

		// Wait for terminal event, client disconnect, or both
		select {
		case <-done:
		case <-ctx.Done():
		}

		unsub()
	}()

	return &abstract.Result{DocumentChannel: eventCh}, nil
}

// timelineEventToMap converts a timeline.TimelineEvent to a JSON-serializable map.
func timelineEventToMap(e timeline.TimelineEvent) map[string]any {
	m := map[string]any{
		"runId":     e.RunID,
		"seq":       e.Seq,
		"timestamp": e.Timestamp,
		"source":    string(e.Source),
		"type":      e.Type,
		"path":      e.Path,
	}
	if e.Payload != nil {
		m["payload"] = e.Payload
	}
	if e.Delta != nil {
		m["delta"] = e.Delta
	}
	return m
}

// ---------------------------------------------------------------------------
// Node registry
// ---------------------------------------------------------------------------

// ListRegisteredNodeKinds returns all registered workflow node type definitions.
//
// @hestia.register(
//   name="system:workflows:registry:list",
//   intent="read",
//   rule="administrator",
//   description="List all registered workflow node kind definitions",
// )
func (s *WorkflowsService) ListRegisteredNodeKinds(_ context.Context, _ abstract.Message, _ *WorkflowRegistryListInput) (*NodeRegistryListView, error) {
	reg := nodekit.Registry()
	out := make([]NodeDefinitionView, 0, len(reg))
	for _, def := range reg {
		out = append(out, nodeDefToView(def))
	}
	return document.New(&NodeRegistryListView{Nodes: out}), nil
}

// GetRegisteredNodeKind returns a single node definition by kind.
//
// @hestia.register(
//   name="system:workflows:registry:get",
//   intent="read",
//   rule="administrator",
//   description="Get a single workflow node kind definition",
//   resource_id="kind",
// )
func (s *WorkflowsService) GetRegisteredNodeKind(_ context.Context, _ abstract.Message, input *WorkflowRegistryGetInput) (*NodeRegistryGetView, error) {
	if input.Kind == "" {
		return nil, common.NewSystemError("WORKFLOW_NODE_KIND_REQUIRED", "kind is required")
	}
	def, ok := nodekit.Get(input.Kind)
	if !ok {
		return nil, common.NewSystemError("WORKFLOW_NODE_KIND_NOT_FOUND", "node kind "+input.Kind+" is not registered")
	}
	return document.New(&NodeRegistryGetView{NodeDefinitionView: nodeDefToView(def)}), nil
}

// NodeHandlesJS returns the raw JS object literal mapping each node kind to its
// handle computation function, matching the contract: new Function("return (" + code + ")")().
// The client evals this once and caches the resulting map.
//
// @hestia.register(
//   name="system:workflows:registry:handles",
//   intent="read",
//   rule="administrator",
//   description="Get the raw JS handle computation functions for all node kinds",
// )
func (s *WorkflowsService) NodeHandlesJS(_ context.Context, _ abstract.Message, _ *WorkflowRegistryHandlesInput) (*WorkflowRegistryHandlesView, error) {
	reg := nodekit.Registry()
	kinds := make([]string, 0, len(reg))
	for k := range reg {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	entries := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		def := reg[kind]
		if def.HandlesJS == "" {
			continue
		}
		entries = append(entries, fmt.Sprintf("%s: %s", strconv.Quote(kind), def.HandlesJS))
	}

	code := "{\n" + strings.Join(entries, ",\n") + "\n}"
	return document.New(&WorkflowRegistryHandlesView{Code: code}), nil
}

// ---------------------------------------------------------------------------
// Runtime convenience methods
// ---------------------------------------------------------------------------

// RuntimeHas checks whether a workflow is registered in the runtime.
//
// @hestia.register(
//   name="system:workflows:runtime:has",
//   intent="read",
//   rule="administrator",
//   description="Check if a workflow is registered in the runtime",
//   resource_id="id",
// )
func (s *WorkflowsService) RuntimeHas(_ context.Context, _ abstract.Message, input *WorkflowRuntimeHasInput) (*WorkflowRuntimeHasView, error) {
	if input.ID == "" {
		return nil, common.NewSystemError("WORKFLOW_ID_REQUIRED", "id is required")
	}
	return document.New(&WorkflowRuntimeHasView{Has: s.runtime.HasWorkflow(input.ID)}), nil
}

// RuntimeListWorkflows returns IDs of all registered (active) workflows.
//
// @hestia.register(
//   name="system:workflows:runtime:list",
//   intent="read",
//   rule="administrator",
//   description="List IDs of all registered (active) workflows",
// )
func (s *WorkflowsService) RuntimeListWorkflows(_ context.Context, _ abstract.Message, _ *WorkflowRuntimeListInput) (*WorkflowRuntimeListView, error) {
	return document.New(&WorkflowRuntimeListView{WorkflowIDs: s.runtime.ListWorkflows()}), nil
}

// RuntimeInvoke directly invokes a registered workflow's trigger.
//
// @hestia.register(
//   name="system:workflows:runtime:invoke",
//   intent="create",
//   rule="administrator",
//   description="Invoke a registered workflow's trigger directly",
// )
func (s *WorkflowsService) RuntimeInvoke(_ context.Context, _ abstract.Message, input *WorkflowRuntimeInvokeInput) (*WorkflowRuntimeInvokeView, error) {
	if input.WorkflowID == "" {
		return nil, common.NewSystemError("WORKFLOW_ID_REQUIRED", "workflow_id is required")
	}
	if input.TriggerID == "" {
		return nil, common.NewSystemError("WORKFLOW_TRIGGER_ID_REQUIRED", "trigger_id is required")
	}
	result := s.runtime.Invoke(input.WorkflowID, input.TriggerID, events.PipelineEvent{
		Payload: input.Payload,
	})
	v := &WorkflowRuntimeInvokeView{
		RunID:  result.RunID,
		Status: result.Status,
		OK:     result.OK,
	}
	if result.Error != nil {
		v.Error = result.Error.Error()
	}
	return document.New(v), nil
}

// RuntimeResume resumes a paused run with the given event payload.
//
// @hestia.register(
//   name="system:workflows:runtime:resume",
//   intent="create",
//   rule="administrator",
//   description="Resume a paused workflow run",
// )
func (s *WorkflowsService) RuntimeResume(_ context.Context, _ abstract.Message, input *WorkflowRuntimeResumeInput) (*WorkflowRuntimeResumeView, error) {
	if input.RunID == "" {
		return nil, common.NewSystemError("WORKFLOW_RUN_ID_REQUIRED", "run_id is required")
	}
	result := s.runtime.Resume(input.RunID, input.Payload)
	v := &WorkflowRuntimeResumeView{
		RunID:  result.RunID,
		Status: result.Status,
		OK:     result.OK,
	}
	if result.Error != nil {
		v.Error = result.Error.Error()
	}
	return document.New(v), nil
}

// nodeDefToView converts a nodekit.NodeDefinition into a wire-safe view.
func nodeDefToView(def nodekit.NodeDefinition) NodeDefinitionView {
	v := NodeDefinitionView{
		Kind:       def.Kind,
		Label:      def.Label,
		Description: def.Description,
		Icon:       def.Icon,
		Scope:      def.Scope,
		Type:       def.Type,
		BodyHandle: def.BodyHandle,
	}
	if len(def.ConfigSchema) > 0 {
		var m map[string]any
		if json.Unmarshal(def.ConfigSchema, &m) == nil {
			v.ConfigSchema = m
		}
	}
	if len(def.Requirements) > 0 {
		reqs := make([]map[string]any, len(def.Requirements))
		for i, r := range def.Requirements {
			b, _ := json.Marshal(r)
			var m map[string]any
			json.Unmarshal(b, &m)
			reqs[i] = m
		}
		v.Requirements = reqs
	}
	return v
}

// toCompilerNodes converts raw JSON maps to compiler.Node structs.
func toCompilerNodes(raw []map[string]any) ([]compiler.Node, error) {
	nodes := make([]compiler.Node, 0, len(raw))
	for i, r := range raw {
		n := compiler.Node{
			ID:       getString(r, "id"),
			Type:     compiler.NodeType(getString(r, "type")),
			ParentID: getString(r, "parentId"),
		}
		// kind lives under data in the client-side graph format
		if n.Kind = getString(r, "kind"); n.Kind == "" {
			if data, ok := r["data"].(map[string]any); ok {
				n.Kind = getString(data, "kind")
			}
		}
		if n.ID == "" {
			return nil, fmt.Errorf("node %d missing id", i)
		}
		if pos, ok := r["position"].(map[string]any); ok {
			n.Position.X = getFloat(pos, "x")
			n.Position.Y = getFloat(pos, "y")
		}
		if cfg, ok := r["config"].(map[string]any); ok {
			n.Config = cfg
		} else if data, ok := r["data"].(map[string]any); ok {
			if cfg, ok := data["config"].(map[string]any); ok {
				n.Config = cfg
			}
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// toCompilerEdges converts raw JSON maps to compiler.Edge structs.
func toCompilerEdges(raw []map[string]any) ([]compiler.Edge, error) {
	edges := make([]compiler.Edge, 0, len(raw))
	for i, r := range raw {
		e := compiler.Edge{
			ID:           getString(r, "id"),
			Source:       getString(r, "source"),
			Target:       getString(r, "target"),
			SourceHandle: getString(r, "sourceHandle"),
			Role:         compiler.EdgeRole(getString(r, "role")),
		}
		if e.Source == "" || e.Target == "" {
			return nil, fmt.Errorf("edge %d missing source or target", i)
		}
		if e.Role == "" {
			e.Role = compiler.EdgeFlow
		}
		edges = append(edges, e)
	}
	return edges, nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return 0
}
