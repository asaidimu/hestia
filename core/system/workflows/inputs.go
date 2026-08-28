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
