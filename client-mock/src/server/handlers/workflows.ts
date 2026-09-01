/**
 * Workflow (hermes) handlers.
 *
 * Definition CRUD is fully persistent. The runtime is a stub executor: a run
 * walks the node graph in topological-ish order (ad-hoc: declaration order),
 * emits a timeline event per node, and settles as `complete`. Events are
 * replayed to SSE subscribers exactly like the real engine.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { clone, sleep, uuid } from "../../util";
import { requirePayload } from "../context";
import { Topics } from "../../bus";
import { newDoc } from "../store";
import type { RunEventRecord, RunRecord } from "../../schema";

export const WORKFLOW_COLLECTION = "_workflow_definitions_";

const REGISTRY_NODES = [
  { kind: "trigger.event", label: "Event Trigger", type: "resource", description: "Starts a pipeline when an event arrives" },
  { kind: "trigger.cron", label: "Cron Trigger", type: "resource", description: "Starts a pipeline on a schedule" },
  { kind: "http.request", label: "HTTP Request", type: "executable", description: "Performs an HTTP request" },
  { kind: "log.message", label: "Log Message", type: "executable", description: "Writes a structured log line" },
  { kind: "data.transform", label: "Transform", type: "executable", description: "JQ-like state transformation" },
  { kind: "flow.condition", label: "Condition", type: "executable", description: "Branches pipeline execution" },
  { kind: "flow.end", label: "End", type: "executable", description: "Terminal node" },
];

export function workflowHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const emitEvent = async (event: RunEventRecord, persist = true) => {
    if (persist) await tables.run_events.put(event);
    deps.bus.emit(Topics.workflowRun(event.run_id), { data: event });
  };

  const countTriggers = (nodes: { data?: Record<string, unknown> }[]) =>
    nodes.filter((n) => {
      const kind = String((n.data as Record<string, unknown>)?.["kind"] ?? "");
      return kind.startsWith("trigger");
    }).length;

  return {
    "system:workflows:definition:compile": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const nodes = (payload["nodes"] ?? []) as { data?: Record<string, unknown> }[];
        const edges = (payload["edges"] ?? []) as unknown[];
        return ok({
          workflow_id: `wf_${uuid().slice(0, 8)}`,
          label: "adhoc",
          triggers: countTriggers(nodes),
          pipelines: edges.length > 0 ? 1 : nodes.length > 0 ? 1 : 0,
        });
      },
    },

    "system:workflows:definition:create": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const doc = await newDoc(WORKFLOW_COLLECTION, {
          name: String(payload["name"] ?? "workflow"),
          description: payload["description"] ?? "",
          nodes: payload["nodes"] ?? [],
          edges: payload["edges"] ?? [],
          tenant_id: ctx.identity!.user["tenant_id"] ?? "root",
          created_at: Date.now(),
          updated_at: Date.now(),
        });
        await tables.documents.put(doc);
        return ok({ id: doc._id_ });
      },
    },

    "system:workflows:definition:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const doc = await tables.documents.get([WORKFLOW_COLLECTION, ctx.args["id"]]);
        if (!doc) throw err.notFound(`workflow definition "${ctx.args["id"]}"`);
        return ok(clone(doc));
      },
    },

    "system:workflows:definition:update": {
      access: "authenticated",
      handler: async (ctx) => {
        const id = ctx.args["id"];
        const current = await tables.documents.get([WORKFLOW_COLLECTION, id]);
        if (!current) throw err.notFound(`workflow definition "${id}"`);
        const payload = requirePayload(ctx);
        const updated = {
          ...current,
          ...payload,
          updated_at: Date.now(),
        };
        await tables.documents.put(updated);
        return noContent();
      },
    },

    "system:workflows:definition:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        const id = ctx.args["id"];
        const current = await tables.documents.get([WORKFLOW_COLLECTION, id]);
        if (!current) throw err.notFound(`workflow definition "${id}"`);
        await tables.documents.delete([WORKFLOW_COLLECTION, id]);
        return noContent();
      },
    },

    "system:workflows:definition:list": {
      access: "authenticated",
      handler: async () => {
        const docs = await tables.documents.getAllByIndex("by_collection", WORKFLOW_COLLECTION);
        return ok(docs.map(clone));
      },
    },

    "system:workflows:runtime:run": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const runId = await startRun(deps, emitEvent, {
          nodes: (payload["nodes"] ?? []) as unknown[],
          label: "adhoc",
        });
        return ok({ run_id: runId });
      },
    },

    "system:workflows:runtime:invoke": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const workflowId = String(payload["workflow_id"] ?? "");
        const doc = await tables.documents.get([WORKFLOW_COLLECTION, workflowId]);
        if (!doc) throw err.notFound(`workflow definition "${workflowId}"`);
        const runId = await startRun(deps, emitEvent, {
          nodes: (doc["nodes"] ?? []) as unknown[],
          label: String(doc["name"] ?? workflowId),
        });
        return ok({ run_id: runId, status: "recording", ok: true });
      },
    },

    "system:workflows:runtime:resume": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const runId = String(payload["run_id"] ?? "");
        const run = await tables.runs.get(runId);
        if (!run) throw err.notFound(`run "${runId}"`);
        if (run.status !== "paused") {
          return ok({ run_id: runId, status: run.status, ok: false, error: "run is not paused" });
        }
        run.status = "recording";
        await tables.runs.put(run);
        if (deps.options.executeWorkflows !== false) {
          void executeRun(deps, run, emitEvent);
        }
        return ok({ run_id: runId, status: "recording", ok: true });
      },
    },

    "system:workflows:runtime:events": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        await emitEvent({
          run_id: `external_${uuid().slice(0, 8)}`,
          seq: Date.now(),
          timestamp: Date.now(),
          source: "api",
          type: String(payload["type"] ?? "event"),
          payload: (payload["payload"] ?? {}) as Record<string, unknown>,
        }, false);
        return noContent();
      },
    },

    "system:workflows:runtime:abort": {
      access: "authenticated",
      handler: async (ctx) => {
        const runId = ctx.args["run_id"];
        const run = await tables.runs.get(runId);
        if (!run) throw err.notFound(`run "${runId}"`);
        run.status = "failed";
        run.end_time = Date.now();
        run.error = "aborted by client";
        await tables.runs.put(run);
        await emitEvent({
          run_id: runId,
          seq: run.event_count + 1,
          timestamp: Date.now(),
          source: "runtime",
          type: "pipeline:failure",
          payload: { error: "aborted by client" },
        });
        return noContent();
      },
    },

    "system:workflows:runtime:has": {
      access: "authenticated",
      handler: async (ctx) => {
        const doc = await tables.documents.get([WORKFLOW_COLLECTION, ctx.args["id"]]);
        return ok({ has: !!doc });
      },
    },

    "system:workflows:runtime:list": {
      access: "authenticated",
      handler: async () => {
        const docs = await tables.documents.getAllByIndex("by_collection", WORKFLOW_COLLECTION);
        return ok({ workflow_ids: docs.map((d) => d._id_) });
      },
    },

    "system:workflows:run:list": {
      access: "authenticated",
      handler: async () => {
        const runs = await tables.runs.getAll();
        return ok(runs.map(publicRun));
      },
    },

    "system:workflows:run:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const run = await tables.runs.get(ctx.args["run_id"]);
        if (!run) throw err.notFound(`run "${ctx.args["run_id"]}"`);
        return ok(publicRun(run));
      },
    },

    "system:workflows:run:outcome": {
      access: "authenticated",
      handler: async (ctx) => {
        const run = await tables.runs.get(ctx.args["run_id"]);
        if (!run) throw err.notFound(`run "${ctx.args["run_id"]}"`);
        return ok({
          ok: run.status === "complete",
          run_id: run.run_id,
          status: run.status,
          final_state: run.final_state ?? {},
          error: run.error,
        });
      },
    },

    "system:workflows:run:events": {
      access: "authenticated",
      handler: async (ctx) => {
        const events = await tables.run_events.getAllByIndex("by_run", ctx.args["run_id"]);
        return ok({ events: events.sort((a, b) => a.seq - b.seq) });
      },
    },

    "system:workflows:run:store": {
      access: "authenticated",
      handler: async (ctx) => {
        const run = await tables.runs.get(ctx.args["run_id"]);
        if (!run) throw err.notFound(`run "${ctx.args["run_id"]}"`);
        return ok({ state: run.final_state ?? {} });
      },
    },

    "system:workflows:registry:list": {
      access: "authenticated",
      handler: async () => ok({ nodes: REGISTRY_NODES }),
    },

    "system:workflows:registry:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const node = REGISTRY_NODES.find((n) => n.kind === ctx.args["kind"]);
        if (!node) throw err.notFound(`node kind "${ctx.args["kind"]}"`);
        return ok(node);
      },
    },

    "system:workflows:registry:handles": {
      access: "authenticated",
      handler: async () => ok({ code: "({})" }),
    },
  };
}

function publicRun(run: RunRecord) {
  return {
    run_id: run.run_id,
    pipeline_id: run.pipeline_id,
    start_time: run.start_time,
    end_time: run.end_time,
    event_count: run.event_count,
    status: run.status,
    metadata: run.metadata,
  };
}

async function startRun(
  deps: ServerDeps,
  emitEvent: (event: RunEventRecord, persist?: boolean) => Promise<void>,
  spec: { nodes: unknown[]; label: string },
): Promise<string> {
  const { tables } = deps;
  const runId = `run_${uuid().slice(0, 12)}`;
  const run: RunRecord = {
    run_id: runId,
    pipeline_id: spec.label,
    start_time: Date.now(),
    event_count: 0,
    status: "recording",
    final_state: {},
  };
  await tables.runs.put(run);

  if (deps.options.executeWorkflows !== false) {
    void executeRun(deps, run, emitEvent, spec.nodes);
  }
  return runId;
}

/** Stub executor: one timeline event per node, then settle. */
async function executeRun(
  deps: ServerDeps,
  run: RunRecord,
  emitEvent: (event: RunEventRecord, persist?: boolean) => Promise<void>,
  nodes: unknown[] = [],
): Promise<void> {
  await sleep(5);
  let seq = 0;
  try {
    await emitEvent({ run_id: run.run_id, seq: ++seq, timestamp: Date.now(), source: "runtime", type: "pipeline:started", payload: {} });

    for (const [index, node] of nodes.entries()) {
      const n = node as { id?: string; data?: Record<string, unknown> };
      const kind = String(n.data?.["kind"] ?? "node");
      await emitEvent({
        run_id: run.run_id,
        seq: ++seq,
        timestamp: Date.now(),
        source: String(n.id ?? `node_${index}`),
        type: `${kind}:success`,
        payload: { index, label: String(n.data?.["label"] ?? kind) },
        delta: {},
      });
      if (run.status === "failed") return; // aborted mid-flight
    }

    await emitEvent({ run_id: run.run_id, seq: ++seq, timestamp: Date.now(), source: "runtime", type: "pipeline:success", payload: {} });

    run.status = "complete";
    run.end_time = Date.now();
    run.event_count = seq;
    await deps.tables.runs.put(run);
  } catch (error) {
    run.status = "failed";
    run.end_time = Date.now();
    run.error = String(error);
    await deps.tables.runs.put(run);
    await emitEvent({ run_id: run.run_id, seq: ++seq, timestamp: Date.now(), source: "runtime", type: "pipeline:failure", payload: { error: String(error) } });
  }
}
