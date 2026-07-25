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

export interface RequestOptions {
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

export interface DispatchInput {
  arguments?: Record<string, string>;
  modifiers?: Record<string, string | string[]>;
  payload?: unknown;
  headers?: Record<string, string>;
  responseType?: ResponseType;
  bodyType?: BodyType;
  signal?: AbortSignal;
  notifyAuthStateChange?: boolean;
}

export interface Transport {
  base(): string;
  prefix(): string;

  ready(): Promise<void>;

  dispatch<T>(name: string, input?: DispatchInput): Promise<HestiaResponse<T>>;

  get<T>(path: string, options?: RequestOptions): Promise<HestiaResponse<T>>;
  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>>;
  patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>>;
  put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>>;
  delete<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>>;
  check<T>(path: string, body?: unknown, options?: RequestOptions): Promise<HestiaResponse<T>>;
  openStream(path: string, handlers: StreamHandlers, options?: StreamOptions): Promise<void>;
}

interface RouteDoc {
  method: string;
  route: string;
  arguments: string[];
}

const ROUTE_TABLE: Record<string, RouteDoc> = {
  "system:core:health:check":                 { method: "GET",    route: "/system/core/health",                   arguments: [] },
  "system:core:heartbeat":                    { method: "GET",    route: "/system/core/heartbeat",                arguments: [] },
  "system:core:capability:list":              { method: "GET",    route: "/system/core/capability",               arguments: [] },
  "system:core:capability:set":               { method: "PATCH",  route: "/system/core/capability",               arguments: [] },
  "system:core:docs:list":                    { method: "GET",    route: "/system/core/docs",                     arguments: [] },
  "system:core:reset":                        { method: "GET",    route: "/system/core/reset",                    arguments: [] },
  "system:auth:session:create":               { method: "POST",   route: "/system/auth/session",                  arguments: [] },
  "system:auth:session:delete":               { method: "DELETE", route: "/system/auth/session",                  arguments: [] },
  "system:auth:user:register":                { method: "POST",   route: "/system/auth/user",                     arguments: [] },
  "system:auth:password:reset":               { method: "POST",   route: "/system/auth/password",                 arguments: [] },
  "system:auth:password:confirm":             { method: "PATCH",  route: "/system/auth/password",                 arguments: [] },
  "system:auth:bootstrap:password:set":       { method: "PATCH",  route: "/system/auth/bootstrap",                arguments: [] },
  "system:collections:collection:list":       { method: "GET",    route: "/system/collections/collection",        arguments: [] },
  "system:collections:collection:get":        { method: "GET",    route: "/system/collections/collection/{name}", arguments: ["name"] },
  "system:collections:collection:create":     { method: "POST",   route: "/system/collections/collection",        arguments: [] },
  "system:collections:collection:delete":     { method: "DELETE", route: "/system/collections/collection/{name}", arguments: ["name"] },
  "system:collections:document:query":        { method: "POST",   route: "/system/collections/document/{name}/query",   arguments: ["name"] },
  "system:collections:document:get":          { method: "GET",    route: "/system/collections/document/{name}/{doc_id}", arguments: ["name", "doc_id"] },
  "system:collections:document:create":       { method: "POST",   route: "/system/collections/document/{name}",   arguments: ["name"] },
  "system:collections:document:update":       { method: "PATCH",  route: "/system/collections/document/{name}/{doc_id}", arguments: ["name", "doc_id"] },
  "system:collections:document:delete":       { method: "DELETE", route: "/system/collections/document/{name}/{doc_id}", arguments: ["name", "doc_id"] },
  "system:blobs:namespace:list":              { method: "POST",   route: "/system/blobs/namespace/query",         arguments: [] },
  "system:blobs:namespace:create":            { method: "POST",   route: "/system/blobs/namespace",               arguments: [] },
  "system:blobs:namespace:delete":            { method: "DELETE", route: "/system/blobs/namespace/{ns}",          arguments: ["ns"] },
  "system:blobs:blob:list":                   { method: "POST",   route: "/system/blobs/blob/{ns}/query",         arguments: ["ns"] },
  "system:blobs:blob:head":                   { method: "POST",   route: "/system/blobs/blob/{ns}/{key}/query",   arguments: ["ns", "key"] },
  "system:blobs:blob:upload":                 { method: "POST",   route: "/system/blobs/blob/{ns}/{key}",          arguments: ["ns", "key"] },
  "system:blobs:blob:download":               { method: "GET",    route: "/system/blobs/blob/{ns}/{key}",          arguments: ["ns", "key"] },
  "system:blobs:blob:delete":                 { method: "DELETE", route: "/system/blobs/blob/{ns}/{key}",          arguments: ["ns", "key"] },
  "system:blobs:blob:update":                 { method: "PATCH",  route: "/system/blobs/blob/{ns}/{key}",          arguments: ["ns", "key"] },
  "system:apikeys:key:list":                  { method: "GET",    route: "/system/apikeys/key",                    arguments: [] },
  "system:apikeys:key:get":                   { method: "GET",    route: "/system/apikeys/key/{key_id}",           arguments: ["key_id"] },
  "system:apikeys:key:create":                { method: "POST",   route: "/system/apikeys/key",                    arguments: [] },
  "system:apikeys:key:update":                { method: "PATCH",  route: "/system/apikeys/key/{key_id}",           arguments: ["key_id"] },
  "system:apikeys:key:delete":                { method: "DELETE", route: "/system/apikeys/key/{key_id}",           arguments: ["key_id"] },
  "system:apikeys:key:rotate":                { method: "POST",   route: "/system/apikeys/key/{key_id}",           arguments: ["key_id"] },
  "system:users:user:query":                  { method: "POST",   route: "/system/users/user/query",               arguments: [] },
  "system:users:user:get":                    { method: "GET",    route: "/system/users/user/{user_id}",            arguments: ["user_id"] },
  "system:users:user:update":                 { method: "PATCH",  route: "/system/users/user/{user_id}",            arguments: ["user_id"] },
  "system:users:user:delete":                 { method: "DELETE", route: "/system/users/user/{user_id}",            arguments: ["user_id"] },
  "system:users:password:change":              { method: "PATCH",  route: "/system/users/password/{user_id}",        arguments: ["user_id"] },
  "system:policies:rule:list":                { method: "GET",    route: "/system/policies/rule",                  arguments: [] },
  "system:policies:rule:get":                 { method: "GET",    route: "/system/policies/rule/{name}",           arguments: ["name"] },
  "system:policies:rule:create":              { method: "POST",   route: "/system/policies/rule/{name}",           arguments: ["name"] },
  "system:policies:rule:update":              { method: "PATCH",  route: "/system/policies/rule/{name}",           arguments: ["name"] },
  "system:policies:rule:delete":              { method: "DELETE", route: "/system/policies/rule/{name}",           arguments: ["name"] },
  "system:policies:rule:validate":            { method: "POST",   route: "/system/policies/rule/check",            arguments: [] },
  "system:policies:reload":                   { method: "GET",    route: "/system/policies/reload",                arguments: [] },
  "system:policies:operation:list":           { method: "GET",    route: "/system/policies/operation",             arguments: [] },
  "system:policies:operation:get":            { method: "GET",    route: "/system/policies/operation/{name}",      arguments: ["name"] },
  "system:policies:policy:list":              { method: "GET",    route: "/system/policies/policy",                arguments: [] },
  "system:policies:policy:create":            { method: "POST",   route: "/system/policies/policy/{name}",         arguments: ["name"] },
  "system:policies:policy:update":            { method: "PATCH",  route: "/system/policies/policy/{name}",         arguments: ["name"] },
};

export class HttpTransport implements Transport {
  private raw: NetworkClient;

  constructor(
    private baseUrl: string,
    private apiPrefix: string,
    private onAuthStateChanged?: () => void,
  ) {
    this.raw = createNetworkClient({
      baseUrl,
      defaultResponseType: "json",
      defaultBodyType: "json",
    });
  }

  private substituteArgs(route: string, args: Record<string, string>): string {
    return route.replace(/\{(\w+)\}/g, (_, key) => {
      if (args[key]) return encodeURIComponent(args[key]);
      return `{${key}}`;
    });
  }

  async dispatch<T>(name: string, input?: DispatchInput): Promise<HestiaResponse<T>> {
    const entry = ROUTE_TABLE[name];

    if (!entry) {
      throw new SystemError({
        code: "ROUTE_NOT_FOUND",
        message: `No registered route for handler: ${name}`,
      });
    }

    const path = this.substituteArgs(entry.route, input?.arguments ?? {});
    const method = entry.method as HttpMethod;
    const options: RequestOptions = {};
    const notifyAuthStateChange = input?.notifyAuthStateChange ?? true;

    if (input?.headers) options.headers = input.headers;
    if (input?.responseType) options.responseType = input.responseType;
    if (input?.bodyType) options.bodyType = input.bodyType;
    if (input?.signal) options.signal = input.signal;

    if (input?.modifiers) {
      const qs = Object.entries(input.modifiers)
        .flatMap(([k, v]) => {
          const vals = Array.isArray(v) ? v : [v];
          return vals.map((vv) => `${encodeURIComponent(k)}=${encodeURIComponent(vv)}`);
        })
        .join("&");
      if (qs) {
        const sep = path.includes("?") ? "&" : "?";
        return this.request<T>(method, `${path}${sep}${qs}`, input?.payload, options, notifyAuthStateChange);
      }
    }

    return this.request<T>(method, path, input?.payload, options, notifyAuthStateChange);
  }

  base() {
    return this.baseUrl;
  }

  prefix(): string {
    return this.apiPrefix;
  }

  async ready(): Promise<void> {
    return;
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
    notifyAuthStateChange = false,
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

    if (res.status === 401 && notifyAuthStateChange) {
      this.onAuthStateChanged?.();
    }

    const errorBody = res.raw ? await parseErrorBody(res.raw) : null;
    throw toSystemError(res, errorBody);
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

// Backward compatibility alias
export const HestiaNetworkClient = HttpTransport;
export type HestiaNetworkClient = HttpTransport;
