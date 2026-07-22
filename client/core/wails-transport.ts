import { Once } from "@asaidimu/utils-sync";
import {
  type Transport,
  HttpTransport,
  HestiaResponse,
  type IdentityProvider,
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
  baseUrl?: string;
  apiPrefix?: string;
  identityProvider?: IdentityProvider;
  onUnauthorized?: () => void;
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

function noHttpError(): never {
  throw new SystemError({
    code: "HTTP_UNAVAILABLE",
    message: "WailsTransport needs baseUrl and identityProvider for path-based requests. Use dispatch() instead.",
  });
}

export class WailsTransport implements Transport {
  private baseUrl = "";
  private apiPrefix = "";
  private init = new Once<void>({ throws: true });
  private config?: WailsTransportConfig;
  private http?: HttpTransport;
  private onUnauthorized?: () => void;

  constructor(config?: WailsTransportConfig) {
    this.config = config;
    this.baseUrl = config?.baseUrl ?? "";
    this.apiPrefix = config?.apiPrefix ?? "/api";
    this.onUnauthorized = config?.onUnauthorized;

    if (config?.baseUrl) {
      this.http = new HttpTransport(
        config.baseUrl,
        this.apiPrefix,
        () => this.onUnauthorized?.(),
      );
    }
  }

  setOnUnauthorized(cb: () => void, provider?: IdentityProvider) {
    this.onUnauthorized = cb;
    if (!this.http) {
      this.http = new HttpTransport(
        this.baseUrl || "http://wails.local",
        this.apiPrefix,
        () => this.onUnauthorized?.(),
      );
    }
  }

  configure(baseUrl: string, apiPrefix: string, provider?: IdentityProvider) {
    this.baseUrl = baseUrl;
    this.apiPrefix = apiPrefix;
    if (!this.http) {
      this.http = new HttpTransport(
        baseUrl,
        apiPrefix,
        () => this.onUnauthorized?.(),
      );
    }
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
    await this.init.do(async () => {
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
    await this.ready();

    if (this.http && (input?.bodyType === "blob" || input?.bodyType === "stream")) {
      return this.http.dispatch<T>(name, input);
    }

    const notify = input?.notifyAuthStateChange ?? true;
    let payload = input?.payload ?? null;
    const modifiers = { ...(input?.modifiers ?? {}) };

    if (input?.headers?.["Content-Type"]) {
      (modifiers as Record<string, string>)["content_type"] = input.headers["Content-Type"];
    }

    const svc = this.resolveService();
    if (svc?.Dispatch) {
      let resp: WailsDispatchResponse;
      try {
        const result = await svc.Dispatch({
          name,
          arguments: input?.arguments ?? {},
          modifiers,
          payload,
        } as WailsDispatchPayload);
        resp = result as unknown as WailsDispatchResponse;
      } catch (err: any) {
        console.error(`[WailsTransport] dispatch "${name}" failed:`, err?.message ?? err);
        throw err instanceof SystemError
          ? err
          : new SystemError({ code: "DISPATCH_ERROR", message: err?.message ?? String(err) });
      }

      if (resp.status >= 400) {
        const errorData = (resp.data as any)?.error;
        const code = errorData?.code ?? "UNKNOWN";
        const message = errorData?.message ?? `dispatch returned status ${resp.status}`;
        console.error(`[WailsTransport] dispatch "${name}" error [${code}]: ${message}`);

        if (resp.status === 401 && notify) {
          this.init = new Once<void>({ throws: true });
          this.onUnauthorized?.();
        }

        throw new SystemError({ code, message });
      }

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
    await this.ready();
    if (!this.http) throw noHttpError();
    return this.http.check<T>(path, body, options);
  }

  async get<T>(
    path: string,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    await this.ready();
    if (!this.http) throw noHttpError();
    return this.http.get<T>(path, options);
  }

  async post<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    await this.ready();
    if (!this.http) throw noHttpError();
    return this.http.post<T>(path, body, options);
  }

  async patch<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    await this.ready();
    if (!this.http) throw noHttpError();
    return this.http.patch<T>(path, body, options);
  }

  async put<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    await this.ready();
    if (!this.http) throw noHttpError();
    return this.http.put<T>(path, body, options);
  }

  async delete<T>(
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    await this.ready();
    if (!this.http) throw noHttpError();
    return this.http.delete<T>(path, body, options);
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
