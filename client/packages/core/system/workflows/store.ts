import type { Document } from "../../core/types"
import { type Transport, type StreamOptions } from "../../core/client"
import type {
  WorkflowDefinition,
  RegisterWorkflowPayload,
  RunWorkflowPayload,
  EmitEventPayload,
  CompiledWorkflow,
  RunMeta,
  RunOutcome,
  TimelineEvent,
  NodeDefinition,
  HandleSpec,
  InvokeWorkflowPayload,
  InvokeWorkflowResult,
  ResumeWorkflowPayload,
  ResumeWorkflowResult,
} from "./types"

// ---------------------------------------------------------------------------
// Handles cache — fetched once, eval'd, stored for synchronous access
// ---------------------------------------------------------------------------

type HandlesMap = Record<string, (config: any) => HandleSpec[]>

let handlesCache: HandlesMap | null = null

/** Synchronous access to the cached handles map. Returns null until fetchHandles() completes. */
export function getCachedHandles(): HandlesMap | null {
  return handlesCache
}

/** Client store for the hermes workflow engine exposed via hestia. */
export class HestiaWorkflowStore {
  constructor(private client: Transport) {}

  // ---------------------------------------------------------------------------
  // Definition operations
  // ---------------------------------------------------------------------------

  /** Compile a workflow graph into a pipeline definition without executing. */
  async compile(nodes: Record<string, any>[], edges: Record<string, any>[]): Promise<CompiledWorkflow> {
    const res = await this.client.dispatch<{ data: CompiledWorkflow }>(
      "system:workflows:definition:compile",
      { payload: { nodes, edges } },
    )
    return res.data?.data!
  }

  /** Store and register a workflow definition. Returns the stored definition ID. */
  async create(payload: RegisterWorkflowPayload): Promise<string> {
    const res = await this.client.dispatch<{ data: { id: string } }>(
      "system:workflows:definition:create",
      { payload },
    )
    return res.data?.data?.id ?? ""
  }

  /** Delete (deregister) a workflow definition. */
  async delete(id: string): Promise<void> {
    await this.client.dispatch("system:workflows:definition:delete", {
      arguments: { id },
    })
  }

  /** Update a workflow definition's fields. */
  async update(id: string, payload: Partial<RegisterWorkflowPayload>): Promise<void> {
    await this.client.dispatch("system:workflows:definition:update", {
      arguments: { id },
      payload,
    })
  }

  /** List all stored workflow definitions. */
  async list(): Promise<Document<WorkflowDefinition>[]> {
    const res = await this.client.dispatch<{ data: Document<WorkflowDefinition>[] }>(
      "system:workflows:definition:list",
    )
    return res.data?.data ?? []
  }

  /** Get a single workflow definition by ID. Returns null if not found. */
  async get(id: string): Promise<Document<WorkflowDefinition> | null> {
    const res = await this.client.dispatch<{ data: Document<WorkflowDefinition> }>(
      "system:workflows:definition:get",
      { arguments: { id } },
    )
    return res.data?.data ?? null
  }

  // ---------------------------------------------------------------------------
  // Runtime operations
  // ---------------------------------------------------------------------------

  /** Compile and execute a workflow graph ad-hoc. Returns the run ID. */
  async run(payload: RunWorkflowPayload): Promise<string> {
    const res = await this.client.dispatch<{ data: { run_id: string } }>(
      "system:workflows:runtime:run",
      { payload },
    )
    return res.data?.data?.run_id ?? ""
  }

  /** Emit a trigger event to the workflow runtime bus. */
  async emitEvent(payload: EmitEventPayload): Promise<void> {
    await this.client.dispatch("system:workflows:runtime:events", { payload })
  }

  /** Abort a running workflow. */
  async abort(runId: string): Promise<void> {
    await this.client.dispatch("system:workflows:runtime:abort", {
      arguments: { run_id: runId },
    })
  }

  // ---------------------------------------------------------------------------
  // Run inspection
  // ---------------------------------------------------------------------------

  /** List all execution runs. */
  async listRuns(): Promise<RunMeta[]> {
    const res = await this.client.dispatch<{ data: RunMeta[] }>(
      "system:workflows:run:list",
    )
    return res.data?.data ?? []
  }

  /** Get metadata for a single run. Returns null if not found. */
  async getRun(runId: string): Promise<RunMeta | null> {
    const res = await this.client.dispatch<{ data: RunMeta }>(
      "system:workflows:run:get",
      { arguments: { run_id: runId } },
    )
    return res.data?.data ?? null
  }

  /** Get the settlement outcome of a run. Returns null if not found. */
  async getOutcome(runId: string): Promise<RunOutcome | null> {
    const res = await this.client.dispatch<{ data: RunOutcome }>(
      "system:workflows:run:outcome",
      { arguments: { run_id: runId } },
    )
    return res.data?.data ?? null
  }

  /** Get timeline events for a run. */
  async getEvents(runId: string): Promise<TimelineEvent[]> {
    const res = await this.client.dispatch<{ data: { events: TimelineEvent[] } }>(
      "system:workflows:run:events",
      { arguments: { run_id: runId } },
    )
    return res.data?.data?.events ?? []
  }

  /** Get the live state of a run. Returns null if not found. */
  async getStore(runId: string): Promise<Record<string, unknown> | null> {
    const res = await this.client.dispatch<{ data: { state: Record<string, unknown> } }>(
      "system:workflows:run:store",
      { arguments: { run_id: runId } },
    )
    return res.data?.data?.state ?? null
  }

  // ---------------------------------------------------------------------------
  // Node registry
  // ---------------------------------------------------------------------------

  /** List all registered workflow node kind definitions. */
  async getRegistry(): Promise<NodeDefinition[]> {
    const res = await this.client.dispatch<{ data: { nodes: NodeDefinition[] } }>(
      "system:workflows:registry:list",
    )
    return res.data?.data?.nodes ?? []
  }

  /** Get a single node definition by kind. */
  async getNodeDefinition(kind: string): Promise<NodeDefinition | null> {
    const res = await this.client.dispatch<{ data: NodeDefinition }>(
      "system:workflows:registry:get",
      { arguments: { kind } },
    )
    return res.data?.data ?? null
  }

  /**
   * Fetch the raw JS handle computation functions from the server,
   * evaluate them, and cache the resulting map.
   * After this call, use getCachedHandles() for synchronous access.
   */
  async fetchHandles(): Promise<HandlesMap> {
    if (handlesCache) return handlesCache
    try {
      const res = await this.client.dispatch<{ data: { code: string } }>(
        "system:workflows:registry:handles",
      )
      const code = res.data?.data?.code
      if (!code) {
        handlesCache = {}
        return handlesCache
      }
      // eslint-disable-next-line no-new-func
      const fn = new Function(`return (${code})`)
      handlesCache = fn() as HandlesMap
      return handlesCache
    } catch {
      handlesCache = {}
      return handlesCache
    }
  }

  // ---------------------------------------------------------------------------
  // Runtime convenience methods
  // ---------------------------------------------------------------------------

  /** Check if a workflow is registered in the runtime. */
  async hasWorkflow(id: string): Promise<boolean> {
    const res = await this.client.dispatch<{ data: { has: boolean } }>(
      "system:workflows:runtime:has",
      { arguments: { id } },
    )
    return res.data?.data?.has ?? false
  }

  /** List IDs of all registered (active) workflows. */
  async listWorkflows(): Promise<string[]> {
    const res = await this.client.dispatch<{ data: { workflow_ids: string[] } }>(
      "system:workflows:runtime:list",
    )
    return res.data?.data?.workflow_ids ?? []
  }

  /** Invoke a registered workflow's trigger directly. */
  async invokeWorkflow(payload: InvokeWorkflowPayload): Promise<InvokeWorkflowResult> {
    const res = await this.client.dispatch<{ data: InvokeWorkflowResult }>(
      "system:workflows:runtime:invoke",
      { payload },
    )
    return res.data?.data as InvokeWorkflowResult
  }

  /** Resume a paused workflow run. */
  async resumeWorkflow(payload: ResumeWorkflowPayload): Promise<ResumeWorkflowResult> {
    const res = await this.client.dispatch<{ data: ResumeWorkflowResult }>(
      "system:workflows:runtime:resume",
      { payload },
    )
    return res.data?.data as ResumeWorkflowResult
  }

  // ---------------------------------------------------------------------------
  // SSE streaming
  // ---------------------------------------------------------------------------

  /**
   * Stream timeline events for a run in real-time via SSE.
   *
   * The stream first replays all existing events, then forwards new events
   * as they arrive from the runtime. It closes automatically on terminal
   * events (pipeline:success, pipeline:failure, pipeline:pause) or when
   * the client calls `abort()`.
   *
   * Returns an `AbortController` — call `abort()` to close the stream.
   */
  streamRun(
    runId: string,
    handlers: {
      onEvent: (event: TimelineEvent) => void
      onDone?: () => void
      onError?: (err: Error) => void
      onOpen?: () => void
    },
    options?: StreamOptions,
  ): AbortController {
    const controller = new AbortController()
    const signal = AbortSignal.any([controller.signal, ...(options?.signal ? [options.signal] : [])])

    this.client
      .openStream(
        `/system/workflows/run/stream/${encodeURIComponent(runId)}`,
        {
          onMessage: (data: string) => {
            try {
              const parsed = JSON.parse(data)
              const event = (parsed.data ?? parsed) as TimelineEvent
              handlers.onEvent(event)
            } catch {
              // Skip malformed events
            }
          },
          onError: handlers.onError,
          onOpen: handlers.onOpen,
          onClose: handlers.onDone,
        },
        { ...options, signal },
      )
      .catch((err) => {
        handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
      })

    return controller
  }
}
