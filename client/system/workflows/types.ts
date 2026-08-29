/** A workflow definition stored in the system. */
export interface WorkflowDefinition {
  _id_: string
  /** Human-readable name for the workflow. */
  name: string
  /** Optional description. */
  description?: string
  /** Workflow graph nodes (serialized JSON). */
  nodes: WorkflowNode[]
  /** Workflow graph edges (serialized JSON). */
  edges: WorkflowEdge[]
  tenant_id?: string
  created_at: number
  updated_at: number
  _metadata_: Record<string, unknown>
}

/** Payload for `definition:create` — register and store a workflow. */
export interface RegisterWorkflowPayload {
  /** Human-readable name. */
  name: string
  /** Optional description. */
  description?: string
  /** Workflow graph nodes. */
  nodes: WorkflowNode[]
  /** Workflow graph edges. */
  edges: WorkflowEdge[]
}

/** Payload for `runtime:run` — compile and execute ad-hoc. */
export interface RunWorkflowPayload {
  /** Workflow graph nodes. */
  nodes: WorkflowNode[]
  /** Workflow graph edges. */
  edges: WorkflowEdge[]
}

/** Payload for `runtime:events` — emit a trigger event. */
export interface EmitEventPayload {
  /** Event type name (must match a registered trigger). */
  type: string
  /** Event payload. */
  payload?: Record<string, unknown>
}

// ---------------------------------------------------------------------------
// Hermes wire types (re-exported for convenience)
// ---------------------------------------------------------------------------

export type HandleType = "source" | "target"
export type HandleKind = "executable" | "resource"
export type NodeType = "executable" | "resource"
export type EdgeRole = "flow" | "dependency" | "placeholder"

export interface HandleSpec {
  type: HandleType
  id: string
  label?: string
  kind?: HandleKind
}

export interface WorkflowNode {
  id: string
  type?: NodeType
  data: Record<string, any>
  parentId?: string
  position: { x: number; y: number }
}

export interface WorkflowEdge {
  id: string
  source: string
  sourceHandle?: string
  target: string
  targetHandle?: string
  data?: { role: EdgeRole }
}

// ---------------------------------------------------------------------------
// Run types
// ---------------------------------------------------------------------------

export type RunStatus = "recording" | "complete" | "failed" | "paused"

/** Metadata for a workflow execution run. */
export interface RunMeta {
  run_id: string
  pipeline_id: string
  start_time: number
  end_time?: number
  event_count: number
  status: RunStatus
  metadata?: Record<string, unknown>
}

/** Settlement status of a run. */
export interface RunOutcome {
  ok: boolean
  run_id: string
  status: string
  final_state?: Record<string, unknown>
  error?: string
}

/** A single timeline event. */
export interface TimelineEvent {
  run_id: string
  seq: number
  timestamp: number
  source: string
  type: string
  payload: Record<string, unknown>
  delta?: Record<string, unknown>
}

/** Compiled workflow view. */
export interface CompiledWorkflow {
  workflow_id: string
  label: string
  triggers: number
  pipelines: number
}

// ---------------------------------------------------------------------------
// Node registry types
// ---------------------------------------------------------------------------

/** A single node kind definition from the workflow node registry. */
export interface NodeDefinition {
  kind: string
  label: string
  description?: string
  icon?: string
  config_schema?: Record<string, unknown>
  requirements?: Record<string, unknown>[]
  scope?: string
  type?: string
  body_handle?: string
}

/** Payload for `runtime:invoke` — invoke a workflow trigger directly. */
export interface InvokeWorkflowPayload {
  workflow_id: string
  trigger_id: string
  payload?: Record<string, unknown>
}

/** Payload for `runtime:resume` — resume a paused run. */
export interface ResumeWorkflowPayload {
  run_id: string
  payload?: Record<string, unknown>
}

/** Result of invoking a workflow trigger. */
export interface InvokeWorkflowResult {
  run_id: string
  status: string
  ok: boolean
  error?: string
}

/** Result of resuming a paused run. */
export interface ResumeWorkflowResult {
  run_id: string
  status: string
  ok: boolean
  error?: string
}
