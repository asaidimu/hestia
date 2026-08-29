package workflows

// Compile
type WorkflowCompileInput struct {
	Nodes []map[string]any `input:"payload.nodes"`
	Edges []map[string]any `input:"payload.edges"`
}

// Register (store definition + register with runtime)
type WorkflowRegisterInput struct {
	Name        string           `input:"payload.name"`
	Description string           `input:"payload.description"`
	Nodes       []map[string]any `input:"payload.nodes"`
	Edges       []map[string]any `input:"payload.edges"`
}

// Deregister
type WorkflowDeregisterInput struct {
	WorkflowID string `input:"payload.workflow_id"`
}

// Run (ad-hoc compile + execute)
type WorkflowRunInput struct {
	Nodes []map[string]any `input:"payload.nodes"`
	Edges []map[string]any `input:"payload.edges"`
}

// Emit event
type WorkflowEventInput struct {
	Type    string         `input:"payload.type"`
	Payload map[string]any `input:"payload.payload"`
}

// Abort
type WorkflowAbortInput struct {
	RunID string `input:"arguments.run_id"`
}

// List runs
type WorkflowRunListInput struct{}

// Get run
type WorkflowRunGetInput struct {
	RunID string `input:"arguments.run_id"`
}

// Get run outcome
type WorkflowRunOutcomeInput struct {
	RunID string `input:"arguments.run_id"`
}

// Get run events
type WorkflowRunEventsInput struct {
	RunID string `input:"arguments.run_id"`
}

// Get run store
type WorkflowRunStoreInput struct {
	RunID string `input:"arguments.run_id"`
}

// List definitions
type WorkflowDefinitionListInput struct{}

// Get definition
type WorkflowDefinitionGetInput struct {
	ID string `input:"arguments.id"`
}

// Delete definition
type WorkflowDefinitionDeleteInput struct {
	ID string `input:"arguments.id"`
}

// Update definition
type WorkflowDefinitionUpdateInput struct {
	ID          string           `input:"arguments.id"`
	Name        string           `input:"payload.name"`
	Description string           `input:"payload.description"`
	Nodes       []map[string]any `input:"payload.nodes"`
	Edges       []map[string]any `input:"payload.edges"`
}

// Node registry: list all registered node kinds
type WorkflowRegistryListInput struct{}

// Node registry: get a single node definition by kind
type WorkflowRegistryGetInput struct {
	Kind string `input:"arguments.kind"`
}

// Node registry: get the raw JS handle computation functions
type WorkflowRegistryHandlesInput struct{}

// Runtime: check if a workflow is registered in the runtime
type WorkflowRuntimeHasInput struct {
	ID string `input:"arguments.id"`
}

// Runtime: list IDs of all registered (active) workflows
type WorkflowRuntimeListInput struct{}

// Runtime: invoke a registered workflow's trigger directly
type WorkflowRuntimeInvokeInput struct {
	WorkflowID string         `input:"payload.workflow_id"`
	TriggerID  string         `input:"payload.trigger_id"`
	Payload    map[string]any `input:"payload.payload"`
}

// Runtime: resume a paused run
type WorkflowRuntimeResumeInput struct {
	RunID   string         `input:"payload.run_id"`
	Payload map[string]any `input:"payload.payload"`
}

// Stream run events via SSE
type WorkflowRunStreamInput struct {
	RunID string `input:"arguments.run_id"`
}
