import type { Document } from "../../core/types"
import { type Transport } from "../../core/client"
import type {
  WorkflowDefinition,
  RegisterWorkflowPayload,
  RunWorkflowPayload,
  EmitEventPayload,
  CompiledWorkflow,
  RunMeta,
  RunOutcome,
  TimelineEvent,
} from "./types"

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
}
