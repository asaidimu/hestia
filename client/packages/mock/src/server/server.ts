/**
 * MockHestiaServer — the IndexedDB-backed emulation of the Hestia backend.
 *
 * Responsibilities:
 *  - route every `system:*` handler name to its handler with the correct
 *    access level (public / authenticated / admin)
 *  - resolve the caller identity (session token or X-API-Key)
 *  - wrap results in the server envelope (`{data, metadata}`)
 *  - write an audit-trail entry for every audited dispatch
 *  - emit realtime events on the internal bus (SSE substitute)
 */

import { EventBus, Topics } from "../bus";
import { openMockDatabase, wipeAllStores, type MockTables, STORES } from "../schema";
import { seedDatabase, type SeedConfig, DEFAULT_SEED } from "../seed";
import { sleep, uuid } from "../util";
import { err } from "../errors";
import {
  checkAccess,
  resolveIdentity,
} from "./auth";
import type { RequestContext, RouteSpec } from "./context";
import type { ServerResponse } from "../envelope";
import type { MockServerOptions, ServerDeps } from "./deps";
import { coreHandlers } from "./handlers/core";
import { userHandlers } from "./handlers/users";
import { collectionHandlers } from "./handlers/collections";
import { apiKeyHandlers } from "./handlers/apikeys";
import { notificationHandlers, settingHandlers } from "./handlers/notifications";
import { policyHandlers } from "./handlers/policies";
import { scheduleHandlers } from "./handlers/schedules";
import { workflowHandlers } from "./handlers/workflows";
import { updateHandlers, logHandlers } from "./handlers/updates";
import { blobHandlers } from "./handlers/blobs";
import { AUDIT_LOG_COLLECTION, newDoc, putDoc } from "./store";

export interface MockHestiaServerConfig {
  database?: string;
  seed?: SeedConfig | false;
  /** Wipe the database before opening (fresh server every time). */
  reset?: boolean;
  options?: MockServerOptions;
}

interface RouteEntry extends RouteSpec {
  name: string;
}

export class MockHestiaServer {
  readonly tables: MockTables;
  readonly bus: EventBus;
  readonly options: MockServerOptions;
  readonly databaseName: string;
  private readonly db: IDBDatabase;
  private routes: Map<string, RouteEntry>;
  private booted: Promise<void>;
  private schedulerTimer: ReturnType<typeof setInterval> | null = null;

  private constructor(db: IDBDatabase, tables: MockTables, bus: EventBus, options: MockServerOptions) {
    this.db = db;
    this.tables = tables;
    this.bus = bus;
    this.options = options;
    this.databaseName = db.name;
    this.routes = new Map();
    this.booted = Promise.resolve();
  }

  /** Open (or attach to) the database, seed on first run, and wire routes. */
  static async create(config: MockHestiaServerConfig = {}): Promise<MockHestiaServer> {
    const { db, tables } = await openMockDatabase(config.database);
    const bus = new EventBus();
    const server = new MockHestiaServer(db, tables, bus, config.options ?? {});

    if (config.reset) {
      await wipeAllStores(db);
    }

    const deps = server.buildDeps();
    server.routes = server.buildRoutes(deps);

    const seedConfig = config.seed === false ? null : (config.seed ?? DEFAULT_SEED);
    await seedDatabase(tables, seedConfig ?? DEFAULT_SEED);

    server.booted = Promise.resolve();
    server.startScheduler();
    return server;
  }

  ready(): Promise<void> {
    return this.booted;
  }

  /** Dispatch a named route — the heart of the mock server. */
  async dispatch(
    route: string,
    input: {
      arguments?: Record<string, string>;
      modifiers?: Record<string, string | string[]>;
      payload?: unknown;
      headers?: Record<string, string>;
    } = {},
  ): Promise<ServerResponse> {
    await this.ready();

    const entry = this.routes.get(route);
    if (!entry) throw err.routeNotFound(route);

    const ctx: RequestContext = {
      route,
      args: input.arguments ?? {},
      modifiers: input.modifiers ?? {},
      payload: input.payload,
      headers: input.headers ?? {},
      identity: null,
      request_id: uuid(),
    };

    ctx.identity = await resolveIdentity(this.tables, ctx);
    checkAccess(entry.access ?? "authenticated", ctx.identity);

    const started = Date.now();
    let response: ServerResponse;
    let failure: unknown = null;
    try {
      response = await entry.handler(ctx);
    } catch (error) {
      failure = error;
      throw error;
    } finally {
      if (!entry.noAudit) {
        try {
          await this.writeAuditEntry(ctx, started, failure);
        } catch {
          // Audit failures must never mask the actual response.
        }
      }
      void failure;
    }
    return response;
  }

  /** Resolve the user id of the persisted current session (used by streams). */
  async currentUserId(): Promise<string | null> {
    const identity = await resolveIdentity(this.tables, { headers: {} });
    return identity?.user._id_ ?? null;
  }

  /** Fire a schedule manually (the mock has no cron daemon). */
  async fireSchedule(scheduleId: string): Promise<void> {
    const doc = await this.tables.documents.get(["_scheduled_messages_", scheduleId]);
    if (!doc || doc["disabled"]) return;
    const message = String(doc["message"] ?? "");
    if (!message) return;

    // Resolve the tiny Go-template subset used by the real server.
    const input = JSON.parse(
      JSON.stringify(doc["input"] ?? {})
        .replaceAll("{{ .schedule._id }}", String(doc["_id_"]))
        .replaceAll("{{ .now }}", new Date().toISOString()),
    );

    try {
      await this.dispatch(message, { payload: input });
      await this.pushLog({ level: "info", msg: `schedule ${scheduleId} dispatched ${message}` });
    } catch (error) {
      await this.pushLog({
        level: "error",
        msg: `schedule ${scheduleId} failed: ${String(error)}`,
      });
    }
  }

  /** Push an application log entry (visible to `appLogs.query`). */
  async pushLog(entry: {
    level: string;
    msg: string;
    caller?: string;
    fields?: Record<string, unknown>;
  }): Promise<void> {
    const record = {
      id: uuid(),
      level: entry.level,
      ts: Date.now(),
      caller: entry.caller ?? "mock",
      msg: entry.msg,
      ...entry.fields,
    };
    await this.tables.logs.put(record);
    this.bus.emit(Topics.logs(), { data: record });
    this.bus.emit(Topics.logs(entry.level), { data: record });
  }

  /** Manually write an audit entry (tests / custom instrumentation). */
  async writeAuditEntry(
    ctx: RequestContext,
    startedAt: number,
    failure: unknown,
  ): Promise<void> {
    const latency = Date.now() - startedAt;
    const segments = ctx.route.split(":");
    const action = segments[segments.length - 1] ?? "other";
    const module = segments[1] ?? "system";

    let operation: string;
    if (module === "auth" && action === "create") operation = "login";
    else if (module === "auth" && action === "delete") operation = "logout";
    else if (action === "create") operation = "create";
    else if (action === "delete") operation = "delete";
    else if (action === "update" || action === "set") operation = "update";
    else if (/^(get|list|query|stream|check|head|progress|status|has)$/.test(action)) operation = "read";
    else if (/^(rotate|elevate|apply|run|invoke|resume|reload|validate|upload|compile|dispatch|abort|export)$/.test(action)) operation = "execute";
    else operation = "other";

    let status: "success" | "failure" = "success";
    let errorMessage: string | undefined;
    if (failure) {
      status = "failure";
      errorMessage = failure instanceof Error ? failure.message : String(failure);
    }

    const doc = await newDoc(AUDIT_LOG_COLLECTION, {
      event_id: uuid(),
      occurred_at: new Date(startedAt).toISOString(),
      recorded_at: new Date().toISOString(),
      trace_id: ctx.request_id,
      request_id: ctx.request_id,
      actor_id: ctx.identity?.user._id_ ?? "anonymous",
      actor_type: ctx.identity ? "user" : "anonymous",
      auth_method: ctx.identity?.auth_method === "api_key" ? "api_key" : ctx.identity ? "password" : "none",
      session_id: ctx.identity?.session_id,
      operation,
      resource_type: module,
      resource_id: Object.values(ctx.args)[0],
      event_name: ctx.route,
      status,
      severity: status === "failure" ? "warning" : "info",
      error_message: errorMessage,
      latency_ms: latency,
      user_agent: "hestia-mock",
      service_name: "hestia-mock",
    });
    await putDoc(this.tables, doc);

    const total = await this.tables.documents.count();
    // Cap the audit trail to keep the mock database bounded.
    if (total > 2000) {
      const entries = await this.tables.documents.getAllByIndex("by_collection", AUDIT_LOG_COLLECTION);
      entries.sort((a, b) => String(a["occurred_at"]).localeCompare(String(b["occurred_at"])));
      const excess = entries.slice(0, entries.length - 500);
      for (const entry of excess) {
        await this.tables.documents.delete([AUDIT_LOG_COLLECTION, entry._id_]);
      }
    }

    this.bus.emit(Topics.audit, { data: doc });
  }

  /** Simulated latency, if configured. */
  async applyLatency(): Promise<void> {
    const latency = this.options.latency;
    if (!latency) return;
    const ms = typeof latency === "function" ? latency() : latency;
    await sleep(ms);
  }

  /** Close the database connection and stop background timers. */
  async close(): Promise<void> {
    this.stopScheduler();
    this.bus.clear();
    this.db.close();
  }

  /** Start auto-ticking `@every` schedules (opt-in via options). */
  private startScheduler(): void {
    const interval = (this.options as { autoTickSchedulesMs?: number }).autoTickSchedulesMs;
    if (!interval) return;
    this.schedulerTimer = setInterval(() => {
      void this.tickSchedules();
    }, interval);
  }

  private stopScheduler(): void {
    if (this.schedulerTimer !== null) {
      clearInterval(this.schedulerTimer);
      this.schedulerTimer = null;
    }
  }

  /** Fire every enabled `@every <n><unit>` schedule whose interval elapsed. */
  async tickSchedules(): Promise<void> {
    const schedules = await this.tables.documents.getAllByIndex("by_collection", "_scheduled_messages_");
    const now = Date.now();
    for (const schedule of schedules) {
      if (schedule["disabled"]) continue;
      const cron = String(schedule["cron"] ?? "");
      const match = /^@every\s+(\d+)(ms|s|m|h)$/.exec(cron);
      if (!match) continue;

      const amount = Number(match[1]);
      const unitMs = { ms: 1, s: 1000, m: 60_000, h: 3_600_000 }[match[2] as "ms" | "s" | "m" | "h"];
      const period = amount * unitMs;

      const lastKey = `schedule_tick_${schedule["_id_"]}`;
      const last = (await this.tables.kv.get(lastKey)) as { v?: number } | undefined;
      if (last?.v && now - last.v < period) continue;
      await this.tables.kv.put({ k: lastKey, v: now });
      await this.fireSchedule(String(schedule["_id_"]));
    }
  }

  private buildDeps(): ServerDeps {
    return {
      tables: this.tables,
      bus: this.bus,
      options: this.options,
      reset: async () => {
        await wipeAllStores(this.db);
        await seedDatabase(this.tables, DEFAULT_SEED);
      },
      pushLog: async (entry) => {
        await this.pushLog(entry);
      },
      setBootstrapped: async (value) => {
        await this.tables.kv.put({ k: "bootstrapped", v: value });
      },
      isBootstrapped: async () => {
        const flag = (await this.tables.kv.get("bootstrapped")) as { v?: boolean } | undefined;
        if (flag?.v != null) return flag.v;
        const users = await this.tables.documents.getAllByIndex("by_collection", "_user_");
        return users.some((u) =>
          Array.isArray(u["permissions"]) && (u["permissions"] as string[]).includes("administrator"),
        );
      },
    };
  }

  private buildRoutes(deps: ServerDeps): Map<string, RouteEntry> {
    const groups = [
      coreHandlers(deps),
      userHandlers(deps),
      collectionHandlers(deps),
      apiKeyHandlers(deps),
      notificationHandlers(deps),
      settingHandlers(deps),
      policyHandlers(deps),
      scheduleHandlers(deps),
      workflowHandlers(deps),
      updateHandlers(deps),
      logHandlers(deps),
      blobHandlers(deps),
    ];

    const map = new Map<string, RouteEntry>();
    for (const group of groups) {
      for (const [name, spec] of Object.entries(group)) {
        map.set(name, { ...spec, name });
      }
    }
    return map;
  }

  /** Route names the mock answers, useful for tests and diagnostics. */
  routeNames(): string[] {
    return [...this.routes.keys()];
  }

  /** Number of live store names (diagnostics). */
  get storeNames(): string[] {
    return Object.keys(STORES);
  }
}
