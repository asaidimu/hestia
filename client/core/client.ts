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

  constructor(
    private baseUrl: string,
    private apiPrefix: string,
    private provider: IdentityProvider,
    private onAuthStateChanged?: () => void,
  ) {
    this.raw = createNetworkClient({
      baseUrl,
      defaultResponseType: "json",
      defaultBodyType: "json",
    });
  }

  base() {
    return this.baseUrl;
  }

  prefix(): string {
    return this.apiPrefix;
  }

  private canonicalPath(path: string): string {
    let cleanPath = path.replace(/^\/+/, "");

    if (this.apiPrefix) {
      const cleanPrefix = this.apiPrefix.replace(/^\/+/, "");

      const prefixRegex = new RegExp(`^${cleanPrefix}/?`);
      cleanPath = cleanPath.replace(prefixRegex, "");

      return `${cleanPrefix}/${cleanPath}`;
    }

    return cleanPath;
  }

  private async request<T>(
    method: HttpMethod,
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<HestiaResponse<T>> {
    const fullPath = this.canonicalPath(path);

    const opts: any = {};
    if (options?.headers) opts.headers = { ...options.headers };
    if (options?.responseType) opts.responseType = options.responseType;
    if (options?.bodyType) opts.bodyType = options.bodyType;
    if (options?.signal) opts.signal = options.signal;

    if (!opts.headers) opts.headers = {};

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

    if (res.status === 401) {
      if (!options?.headers?.["X-API-Key"]) {
        await this.provider.clear()
        this.onAuthStateChanged?.();
      }
      throw new SystemError({
        code: "AUTH-002-UNAUTH",
        message: "Session expired",
      });
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

  async openStream(
    path: string,
    handlers: StreamHandlers,
    options?: StreamOptions,
  ): Promise<void> {
    const headers: Record<string, string> = {
      Accept: "text/event-stream",
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

    if (response.status === 401) {
      this.onAuthStateChanged?.();
      handlers.onError?.(
        new SystemError({
          code: "AUTH-002-UNAUTH",
          message: "Session expired",
        }),
      );
      return;
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
  }
}
