import { Once } from "@asaidimu/utils-sync";
import {
  type Transport,
  HestiaResponse,
  type RequestOptions,
  type StreamHandlers,
  type StreamOptions,
  type DispatchInput,
} from "./client";
import { SystemError } from "@asaidimu/utils-error";

interface WailsDispatchPayload {
  name: string;
  arguments: Record<string, string>;
  modifiers: Record<string, string | string[]>;
  payload: unknown;
}

interface WailsDispatchResponse {
  data: unknown;
  status: number;
}

declare global {
  interface Window {
    go?: Record<
      string,
      Record<string, Record<string, (...args: unknown[]) => Promise<unknown>>>
    >;
    runtime?: {
      Call: (method: string, ...args: unknown[]) => Promise<unknown>;
      EventsOn: (event: string, callback: (...args: unknown[]) => void) => void;
      EventsOff: (event: string) => void;
    };
  }
}

export interface WailsTransportConfig {
  pkg: string;
  struct: string;
}

function findService(): Record<
  string,
  (...args: unknown[]) => Promise<unknown>
> | null {
  try {
    // @ts-ignore
    const go = window.go;
    if (!go) return null;
    for (const pkg of Object.values(go)) {
      for (const svc of Object.values(pkg as any)) {
        // @ts-ignore
        if (svc?.Dispatch) return svc;
      }
    }
    return null;
  } catch {
    return null;
  }
}

export class WailsTransport implements Transport {
  private baseUrl = "";
  private apiPrefix = "";
  private init = new Once<void>({ throws: true });
  private config?: WailsTransportConfig;

  constructor(config?: WailsTransportConfig) {
    this.config = config;
  }

  private resolveService(): Record<
    string,
    (...args: unknown[]) => Promise<unknown>
  > | null {
    if (this.config) {
      try {
        // @ts-ignore
        return window.go?.[this.config.pkg]?.[this.config.struct] ?? null;
      } catch {
        return null;
      }
    }
    return findService();
  }

  base(): string {
    return this.baseUrl;
  }

  prefix(): string {
    return this.apiPrefix;
  }

  /**
   * Polls window.go until Wails bindings are available.
   * Concurrent callers share the single initialization attempt.
   */
  async ready(timeoutMs = 10_000): Promise<void> {
    return void this.init.do(async () => {
      const startTime = Date.now();

      while (!this.resolveService()) {
        if (Date.now() - startTime > timeoutMs) {
          throw new SystemError({
            code: "TRANSPORT_UNAVAILABLE",
            message: `Timed out after ${timeoutMs}ms waiting for Wails bindings on window.go. Are you running inside a Wails desktop window?`,
          });
        }
        await new Promise((resolve) => setTimeout(resolve, 50));
      }
    }, timeoutMs);
  }

  async dispatch<T>(
    name: string,
    input?: DispatchInput,
  ): Promise<HestiaResponse<T>> {
    // Automatically wait for binding resolution before dispatching
    await this.ready();

    const svc = this.resolveService();
    if (svc?.Dispatch) {
      const result = await svc.Dispatch({
        name,
        arguments: input?.arguments ?? {},
        modifiers: input?.modifiers ?? {},
        payload: input?.payload ?? null,
      } as WailsDispatchPayload);

      const resp = result as unknown as WailsDispatchResponse;
      return new HestiaResponse(resp.data as T, resp.status);
    }

    // @ts-ignore
    const available = window.go ? Object.keys(window.go).join(", ") : "none";
    throw new SystemError({
      code: "TRANSPORT_UNAVAILABLE",
      message: `No Wails Dispatch binding found on window.go. Available packages: ${available}.`,
    });
  }

  async check<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    return this.request<T>("POST", `${path}/check`, body, options);
  }

  async get<T>(
    path: string,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    return this.request<T>("GET", path, undefined, options);
  }

  async post<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    return this.request<T>("POST", path, body, options);
  }

  async patch<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    return this.request<T>("PATCH", path, body, options);
  }

  async put<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    return this.request<T>("PUT", path, body, options);
  }

  async delete<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    return this.request<T>("DELETE", path, body, options);
  }

  private async request<T>(
    _method: string,
    _path: string,
    _body?: unknown,
    _options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    throw new SystemError({
      code: "USE_DISPATCH",
      message:
        "WailsTransport does not support path-based requests. Use dispatch(name, input) instead.",
    });
  }

  async openStream(
    path: string,
    handlers: StreamHandlers,
    _options?: StreamOptions,
  ): Promise<void> {
    const svc = this.resolveService();
    try {
      if (!svc?.Stream) {
        // @ts-ignore
        const available = window.go
          ? // @ts-ignore
            Object.keys(window.go).join(", ")
          : "none";
        throw new SystemError({
          code: "TRANSPORT_UNAVAILABLE",
          message: `No Wails Stream binding found. Available packages: ${available}`,
        });
      }

      const streamData = (await svc.Stream(path)) as string;
      const eventName = `hestia:stream:${path}`;
      const onData = (data: unknown) => {
        handlers.onMessage(
          typeof data === "string" ? data : JSON.stringify(data),
        );
      };

      // @ts-ignore
      window.runtime?.EventsOn(eventName, onData);
      handlers.onOpen?.();

      if (streamData) {
        handlers.onMessage(streamData);
      }

      handlers.onClose?.();
    } catch (err) {
      handlers.onError?.(err instanceof Error ? err : new Error(String(err)));
      handlers.onClose?.();
    }
  }
}
