import {
  createNetworkClient,
  type NetworkClient,
  type ApiResponse,
} from "@asaidimu/network-client";
import { SystemError } from "@asaidimu/utils-error";
import { parseErrorBody, toSystemError } from "./errors";
import { UserIdentity } from "../system/identity/types";

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

export class HestiaNetworkClient {
  private raw: NetworkClient;
  private refreshing: Promise<void> | null = null;
  private refreshFailed = false;

  constructor(
    private baseUrl: string,
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
  async storeTokens(access: string, refresh?: string): Promise<void> {
    this.refreshFailed = false;
    await this.tokens.setTokens(access, refresh ?? "");
  }

  private   async refreshToken(): Promise<void> {
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
    }>("/system/auth/session", body);

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
    const opts: any = {};
    if (options?.headers) opts.headers = options.headers;
    if (options?.responseType) opts.responseType = options.responseType;
    if (options?.bodyType) opts.bodyType = options.bodyType;
    if (options?.signal) opts.signal = options.signal;

    let res: ApiResponse<T>;

    if (method === "GET") {
      res = await this.raw.get<T>(path, opts);
    } else {
      const bodyOpts = options?.bodyType
        ? { type: options.bodyType as BodyType }
        : undefined;
      res = (await (this.raw as any)[method.toLowerCase()](
        path,
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

      // If a previous refresh already failed, don't loop — the tokens
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
        retryOpts.headers = { ...(opts.headers ?? {}), Authorization: `Bearer ${newAccess}` };
      }

      if (method === "GET") {
        res = await this.raw.get<T>(path, retryOpts);
      } else {
        const bodyOpts = options?.bodyType
          ? { type: options.bodyType as BodyType }
          : undefined;
        res = (await (this.raw as any)[method.toLowerCase()](
          path,
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
}
