import {
  createNetworkClient,
  type NetworkClient,
  type ApiResponse,
} from "@asaidimu/network-client";
import { SystemError } from "@asaidimu/utils-error";
import { parseErrorBody, toSystemError } from "./errors";
import type { UserIdentity } from "../system/identity/types";

export interface IdentityProvider {
  identity(): UserIdentity | null;
  token(key: "access" | "refresh"): string | null;
  setTokens(access: string, refresh: string): Promise<void>;
  setIdentity(id: UserIdentity | null): Promise<void>;
  clear(): Promise<void>;
}

export class HestiaResponse<T> {
  constructor(
    public readonly data: T,
    public readonly status: number,
  ) {}
}

type HttpMethod = "GET" | "POST" | "PATCH" | "DELETE" | "PUT";
type BodyType = "json" | "form" | "text" | "blob" | "stream" | "auto";
type ResponseType =
  | "json"
  | "text"
  | "blob"
  | "arrayBuffer"
  | "formData"
  | "auto";

interface RequestOptions {
  headers?: Record<string, string>;
  responseType?: ResponseType;
  bodyType?: BodyType;
  signal?: AbortSignal;
}

// Handlers driving a live server-sent-events stream opened via
// HestiaNetworkClient.openStream(). onMessage fires once per parsed SSE
// "data:" payload (still a raw string — callers decide whether/how to
// JSON.parse it); onOpen/onClose/onError report connection lifecycle.
export interface StreamHandlers {
  onMessage: (data: string) => void;
  onError?: (err: Error) => void;
  onOpen?: () => void;
  onClose?: () => void;
}

export interface StreamOptions {
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

export class HestiaNetworkClient {
  private raw: NetworkClient;
  private refreshing: Promise<void> | null = null;
  private refreshFailed = false;

  constructor(
    private baseUrl: string,
    private apiPrefix: string,
    private tokens: IdentityProvider,
    private onAuthStateChanged?: () => void,
  ) {
    this.raw = createNetworkClient({
      baseUrl,
      defaultResponseType: "json",
      defaultBodyType: "json",
      interceptors: {
        request: [
          (ctx) => {
            const access = this.tokens.token("access");
            if (access) {
              ctx.headers["Authorization"] = `Bearer ${access}`;
            }
            return ctx;
          },
        ],
      },
    });
  }

  base() {
    return this.baseUrl;
  }

  prefix(): string {
    return this.apiPrefix;
  }

  // Single entry-point for all path manipulation (prefix handling).
  // Returns the relative path that should be appended to the base URL.
  private canonicalPath(path: string): string {
    // 1. Strip any leading slashes so we work with a clean base (e.g., "api/api/system" or "/api/system")
    let cleanPath = path.replace(/^\/+/, "");

    if (this.apiPrefix) {
      // 2. Strip leading slashes from the prefix too, just to be safe and consistent
      const cleanPrefix = this.apiPrefix.replace(/^\/+/, "");

      // 3. If the path already starts with the prefix followed by a slash (or is exactly the prefix),
      // remove it so we don't double up.
      const prefixRegex = new RegExp(`^${cleanPrefix}/?`);
      cleanPath = cleanPath.replace(prefixRegex, "");

      // 4. Combine them cleanly
      return `${cleanPrefix}/${cleanPath}`;
    }

    return cleanPath;
  }

  async storeTokens(access: string, refresh?: string): Promise<void> {
    this.refreshFailed = false;
    await this.tokens.setTokens(access, refresh ?? "");
  }

  private async refreshToken(): Promise<void> {
    if (this.refreshing) return this.refreshing;

    this.refreshing = this.doRefresh();
    try {
      await this.refreshing;
    } finally {
      this.refreshing = null;
      this.onAuthStateChanged ? this.onAuthStateChanged() : undefined;
    }
  }

  private async doRefresh(): Promise<void> {
    const refresh = this.tokens.token("refresh");
    const body = refresh ? { refresh_token: refresh } : {};
    const res = await this.raw.patch<{
      data: { token: { access: string; refresh: string } };
    }>(this.canonicalPath("/system/auth/session"), body);

    if (!res.success || !res.data) {
      this.tokens.clear();
      this.refreshFailed = true;
      throw new SystemError({
        code: "AUTH-002-UNAUTH",
        message: "Token refresh failed",
      });
    }

    const { access, refresh: newRefresh } = res.data.data.token;
    this.refreshFailed = false;
    await this.tokens.setTokens(access, newRefresh);
  }

  private async request<T>(
    method: HttpMethod,
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    const fullPath = this.canonicalPath(path);

    const opts: any = {};
    if (options?.headers) opts.headers = options.headers;
    if (options?.responseType) opts.responseType = options.responseType;
    if (options?.bodyType) opts.bodyType = options.bodyType;
    if (options?.signal) opts.signal = options.signal;

    let res: ApiResponse<T>;

    if (method === "GET") {
      res = await this.raw.get<T>(fullPath, opts);
    } else {
      const bodyOpts = options?.bodyType
        ? { type: options.bodyType as BodyType }
        : undefined;
      res = (await (this.raw as any)[method.toLowerCase()](
        fullPath,
        body,
        opts,
        bodyOpts,
      )) as ApiResponse<T>;
    }

    if (res.success || res.status === 204) {
      return new HestiaResponse(res.data as T, res.status);
    }

    if (
      (res.status === 401 || res.status === 403) &&
      !path.includes("/system/auth/token") &&
      !path.includes("/system/auth/session")
    ) {
      // Issue B: API-key requests should never trigger a JWT refresh loop.
      if (options?.headers?.["X-API-Key"]) {
        throw new SystemError({
          code: "AUTH-003-APIKEY",
          message: "API key is invalid or expired",
        });
      }

      // If a previous refresh already failed, don’t loop — the tokens
      // (including any cookie fallback) are known to be dead.
      if (this.refreshFailed) {
        this.tokens.clear();
        this.onAuthStateChanged?.();
        throw new SystemError({
          code: "AUTH-002-UNAUTH",
          message: "Session expired",
        });
      }

      try {
        await this.refreshToken();
      } catch {
        this.tokens.clear();
        this.onAuthStateChanged?.();
        throw new SystemError({
          code: "AUTH-002-UNAUTH",
          message: "Session expired",
        });
      }

      // Issue A: Read the new access token and inject it explicitly so that
      // an async setTokens (e.g. IndexedDB write) cannot race the interceptor.
      const newAccess = this.tokens.token("access");
      const retryOpts: any = { ...opts };
      if (newAccess) {
        retryOpts.headers = {
          ...(opts.headers ?? {}),
          Authorization: `Bearer ${newAccess}`,
        };
      }

      if (method === "GET") {
        res = await this.raw.get<T>(fullPath, retryOpts);
      } else {
        const bodyOpts = options?.bodyType
          ? { type: options.bodyType as BodyType }
          : undefined;
        res = (await (this.raw as any)[method.toLowerCase()](
          fullPath,
          body,
          retryOpts,
          bodyOpts,
        )) as ApiResponse<T>;
      }

      if (res.success || res.status === 204) {
        return new HestiaResponse(res.data as T, res.status);
      }
    }

    const errorBody = res.raw ? await parseErrorBody(res.raw) : null;
    throw toSystemError(res, errorBody);
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

  // Opens an authenticated, long-lived GET request against a
  // "text/event-stream" endpoint (e.g. system:audit:log:stream) and parses
  // standard SSE framing ("data: ...\n\n") off the raw response body.
  //
  // Bypasses `raw` deliberately: the wrapped network-client's request/
  // response cycle is built around single JSON responses, not an
  // indefinitely-open ReadableStream, so this talks to `fetch` directly.
  // On a 401 it will attempt exactly one token refresh and reconnect once
  // before giving up — mirroring `request()`'s retry behavior, but capped
  // at a single attempt since a stream that keeps failing auth shouldn’t
  // loop indefinitely in the background.
  async openStream(
    path: string,
    handlers: StreamHandlers,
    options?: StreamOptions,
  ): Promise<void> {
    const attempt = async (isRetry: boolean): Promise<void> => {
      const access = this.tokens.token("access");
      const isApiKeyAuth = !!options?.headers?.["X-API-Key"];
      const headers: Record<string, string> = {
        Accept: "text/event-stream",
        ...(access && !isApiKeyAuth
          ? { Authorization: `Bearer ${access}` }
          : {}),
        ...(options?.headers ?? {}),
      };

      const url = `${this.baseUrl.replace(/\/+$/, "")}/${this.canonicalPath(path)}`;

      let response: Response;
      try {
        response = await fetch(url, {
          method: "GET",
          headers,
          signal: options?.signal,
        });
      } catch (err) {
        if (err instanceof Error && err.name === "AbortError") {
          handlers.onClose?.();
          return;
        }
        handlers.onError?.(err instanceof Error ? err : new Error(String(err)));
        return;
      }

      if (response.status === 401 && !isRetry && !isApiKeyAuth) {
        try {
          await this.refreshToken();
        } catch {
          this.tokens.clear();
          this.onAuthStateChanged?.();
          handlers.onError?.(
            new SystemError({
              code: "AUTH-002-UNAUTH",
              message: "Session expired",
            }),
          );
          return;
        }
        return attempt(true);
      }

      if (!response.ok || !response.body) {
        handlers.onError?.(
          new Error(`Stream request failed with status ${response.status}`),
        );
        return;
      }

      handlers.onOpen?.();

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });

          let separatorIndex = buffer.indexOf("\n\n");
          while (separatorIndex !== -1) {
            const rawEvent = buffer.slice(0, separatorIndex);
            buffer = buffer.slice(separatorIndex + 2);

            const dataLines = rawEvent
              .split("\n")
              .filter((line) => line.startsWith("data:"))
              .map((line) => line.slice(5).trim());

            if (dataLines.length > 0) {
              handlers.onMessage(dataLines.join("\n"));
            }

            separatorIndex = buffer.indexOf("\n\n");
          }
        }
      } catch (err) {
        if (!(err instanceof Error && err.name === "AbortError")) {
          handlers.onError?.(
            err instanceof Error ? err : new Error(String(err)),
          );
        }
      } finally {
        handlers.onClose?.();
      }
    };

    return attempt(false);
  }
}
