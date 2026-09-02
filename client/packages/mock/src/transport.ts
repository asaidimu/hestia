/**
 * IndexedDbTransport — implements the Hestia `Transport` interface on top of
 * `MockHestiaServer` (IndexedDB). Drop-in replacement for `HttpTransport`:
 *
 * ```ts
 * const api = new HestiaClient({ baseUrl: "idb://hestia-mock", transport })
 * ```
 *
 * Implements the full surface: named dispatch, raw path methods (the audit
 * and app-log stores use `client.post(path, ...)` directly), SSE-style
 * `openStream`, 401 → onUnauthorized notification, and simulated latency.
 */

import { HestiaResponse } from "@asaidimu/hestia";
import { SystemError } from "@asaidimu/utils-error";
import type {
  DispatchInput,
  RequestOptions,
  StreamHandlers,
  StreamOptions,
  Transport,
} from "@asaidimu/hestia";
import { MockHestiaServer } from "./server/server";
import { statusForError } from "./errors";
import { MOCK_ROUTE_TABLE } from "./routes";
import { Topics } from "./bus";

export interface IndexedDbTransportOptions {
  baseUrl?: string;
  apiPrefix?: string;
  /** Invoked when a dispatch fails with 401 while auth-state notification is enabled. */
  onUnauthorized?: () => void;
}

interface ResolvedPath {
  route: string;
  args: Record<string, string>;
}

export class IndexedDbTransport implements Transport<string> {
  private baseUrl: string;
  private apiPrefix: string;
  private onUnauthorizedCb?: () => void;

  constructor(
    private readonly server: MockHestiaServer,
    options: IndexedDbTransportOptions = {},
  ) {
    this.baseUrl = options.baseUrl ?? "idb://hestia-mock";
    this.apiPrefix = options.apiPrefix ?? "/api";
    this.onUnauthorizedCb = options.onUnauthorized;
  }

  setOnUnauthorized(cb: () => void): void {
    this.onUnauthorizedCb = cb;
  }

  base(): string {
    return this.baseUrl;
  }

  prefix(): string {
    return this.apiPrefix;
  }

  routeUrl(name: string, args?: Record<string, string>): string {
    const entry = MOCK_ROUTE_TABLE[name];
    if (!entry) {
      throw new SystemError({
        code: "ROUTE_NOT_FOUND",
        message: `No registered route for handler: ${name}`,
      });
    }
    const base = this.baseUrl.replace(/\/+$/, "");
    return `${base}${this.substituteArgs(entry.route, args ?? {})}`;
  }

  async ready(): Promise<void> {
    await this.server.ready();
  }

  async dispatch<T>(name: string, input?: DispatchInput): Promise<HestiaResponse<T>> {
    await this.server.applyLatency();

    const notify = input?.notifyAuthStateChange ?? true;
    try {
      const response = await this.server.dispatch(name, {
        arguments: input?.arguments,
        modifiers: input?.modifiers,
        payload: input?.payload,
        headers: input?.headers,
      });

      if (response.status === 204 || response.body === undefined) {
        return new HestiaResponse<T>(undefined as T, 204);
      }

      // download-style payloads: raw bytes surfaced as a Blob for
      // responseType "blob" (bypasses the envelope, like the HTTP transport).
      if (input?.responseType === "blob" && response.body.data instanceof Uint8Array) {
        const contentType = (response.body.metadata as Record<string, unknown> | undefined)?.["content_type"];
        const blob = new Blob([response.body.data as unknown as BlobPart], {
          type: (contentType as string) ?? "application/octet-stream",
        });
        return new HestiaResponse<T>(blob as unknown as T, response.status);
      }

      // Everything else: the full server envelope (`res.data.data`,
      // `res.data.metadata.page`) exactly as HttpTransport exposes it.
      return new HestiaResponse<T>(response.body as T, response.status);
    } catch (error) {
      const status = statusForError(error);
      if (status === 401 && notify) {
        this.onUnauthorizedCb?.();
      }
      throw error;
    }
  }

  // ── Path-based API (used directly by the audit + app-log stores) ──────────

  async get<T>(path: string, options?: RequestOptions): Promise<HestiaResponse<T>> {
    return await this.dispatchPath<T>("GET", path, undefined, options);
  }

  async post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>> {
    return await this.dispatchPath<T>("POST", path, body, options);
  }

  async patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>> {
    return await this.dispatchPath<T>("PATCH", path, body, options);
  }

  async put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>> {
    return await this.dispatchPath<T>("PUT", path, body, options);
  }

  async delete<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>> {
    return await this.dispatchPath<T>("DELETE", path, body, options);
  }

  async check<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>> {
    return await this.dispatchPath<T>("POST", `${path}/check`, body, options);
  }

  private async dispatchPath<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    await this.server.applyLatency();
    const resolved = this.resolvePath(method, path);
    if (!resolved) {
      throw new SystemError({
        code: "ROUTE_NOT_FOUND",
        message: `No registered route matches ${method} ${path}`,
      });
    }
    return await this.dispatch<T>(resolved.route, {
      arguments: resolved.args,
      payload: body,
      headers: options?.headers,
      responseType: options?.responseType,
      bodyType: options?.bodyType,
      signal: options?.signal,
    });
  }

  private resolvePath(method: string, rawPath: string): ResolvedPath | null {
    // Strip the api prefix and any query string.
    let path = rawPath.split("?")[0] ?? rawPath;
    if (this.apiPrefix) {
      const prefix = this.apiPrefix.replace(/\/+$/, "");
      path = path.replace(new RegExp(`^${prefix}/?`), "/");
    }
    if (!path.startsWith("/")) path = `/${path}`;

    const segments = path.split("/").filter(Boolean);

    for (const [route, entry] of Object.entries(MOCK_ROUTE_TABLE)) {
      if (entry.method !== method) continue;
      const template = entry.route.split("/").filter(Boolean);
      if (template.length !== segments.length) continue;

      const args: Record<string, string> = {};
      let matched = true;
      for (let i = 0; i < template.length; i++) {
        const tpl = template[i]!;
        const actual = segments[i]!;
        if (tpl.startsWith("{") && tpl.endsWith("}")) {
          args[tpl.slice(1, -1)] = decodeURIComponent(actual);
        } else if (tpl !== actual) {
          matched = false;
          break;
        }
      }
      if (matched) return { route, args };
    }
    return null;
  }

  private substituteArgs(route: string, args: Record<string, string>): string {
    return route.replace(/\{(\w+)\}/g, (_, key) =>
      args[key] !== undefined ? encodeURIComponent(args[key]) : `{${key}}`,
    );
  }

  // ── Streams (SSE emulation) ───────────────────────────────────────────────

  /**
   * Subscribes to a server event stream. Message payloads are rendered as
   * SSE `data:` frames exactly like the Go transport (`data: {"data": ...}\n\n`),
   * and delivered through `handlers.onMessage` as the JSON string.
   */
  async openStream(path: string, handlers: StreamHandlers, options?: StreamOptions): Promise<void> {
    await this.server.ready();

    if (options?.signal?.aborted) {
      handlers.onClose?.();
      return;
    }

    const cleanPath = (path.split("?")[0] ?? path).replace(/^\/+/, "").replace(/^(api\/)?/, "/");
    const query = new URLSearchParams(path.split("?")[1] ?? "");

    let unsubscribe: (() => void) | null = null;
    let closeRequested = false;
    let notifyClosed: () => void = () => undefined;
    const closed = new Promise<void>((resolve) => (notifyClosed = resolve));

    const close = () => {
      if (closeRequested) return;
      closeRequested = true;
      unsubscribe?.();
      unsubscribe = null;
      handlers.onClose?.();
      notifyClosed();
    };

    options?.signal?.addEventListener("abort", close, { once: true });

    try {
      if (cleanPath === "/system/notifications/notification/stream") {
        const userId = await this.server.currentUserId();
        const topic = userId ? Topics.notifications(userId) : Topics.notificationsAll;
        unsubscribe = this.server.bus.subscribe(topic, (payload) => {
          handlers.onMessage(JSON.stringify(payload));
        });
      } else if (cleanPath === "/system/audit/log/stream") {
        unsubscribe = this.server.bus.subscribe(Topics.audit, (payload) => {
          handlers.onMessage(JSON.stringify(payload));
        });
      } else if (cleanPath === "/system/logs/stream") {
        const level = query.get("level") ?? undefined;
        unsubscribe = this.server.bus.subscribe(Topics.logs(level), (payload) => {
          handlers.onMessage(JSON.stringify(payload));
        });
      } else if (cleanPath.startsWith("/system/workflows/run/stream/")) {
        const runId = cleanPath.split("/").filter(Boolean).pop()!;
        unsubscribe = await this.subscribeWorkflowRun(runId, handlers, close);
        if (!unsubscribe) return; // already closed during replay
      } else {
        handlers.onError?.(
          new Error(`Stream request failed: no stream endpoint matches ${path}`),
        );
        handlers.onClose?.();
        return;
      }

      handlers.onOpen?.();

      // Hold the "connection" open — exactly like an SSE response — until the
      // caller aborts (or a terminal workflow event closes it).
      await closed;
    } catch (error) {
      handlers.onError?.(error instanceof Error ? error : new Error(String(error)));
      close();
    }
  }

  /** Replay persisted run events, then forward live ones. Returns null when the stream closed during replay. */
  private async subscribeWorkflowRun(
    runId: string,
    handlers: StreamHandlers,
    close: () => void,
  ): Promise<() => void> {
    const TERMINAL = new Set(["pipeline:success", "pipeline:failure", "pipeline:pause"]);

    const existing = await this.server.tables.run_events.getAllByIndex("by_run", runId);
    existing.sort((a, b) => a.seq - b.seq);
    for (const event of existing) {
      handlers.onMessage(JSON.stringify({ data: event }));
      if (TERMINAL.has(event.type)) {
        close();
        return () => undefined;
      }
    }

    return this.server.bus.subscribe(Topics.workflowRun(runId), (payload) => {
      handlers.onMessage(JSON.stringify(payload));
      const type = (payload as { data?: { type?: string } })?.data?.type;
      if (type && TERMINAL.has(type)) {
        setTimeout(close, 0);
      }
    });
  }
}
