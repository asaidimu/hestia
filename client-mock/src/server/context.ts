/**
 * Request context and dependency types shared by all route handlers.
 */

import type { StoredDocument } from "../schema";
import type { ServerResponse } from "../envelope";

export interface ResolvedIdentity {
  /** Full `_user_` document (including the password hash — handlers must not leak it). */
  user: StoredDocument;
  auth_method: "password" | "api_key" | "bootstrap_key";
  session_id?: string;
  api_key_id?: string;
  is_admin: boolean;
}

export interface RequestContext {
  /** Route name, e.g. `system:collections:document:create`. */
  route: string;
  /** Route arguments (`{name}`, `{doc_id}`, ...). */
  args: Record<string, string>;
  /** Query-string style modifiers. */
  modifiers: Record<string, string | string[]>;
  /** JSON or binary payload. */
  payload: unknown;
  /** Request headers (case-insensitive lookup via `header()`). */
  headers: Record<string, string>;
  /** Resolved identity, or null for anonymous requests. */
  identity: ResolvedIdentity | null;
  request_id: string;
}

export type AccessLevel = "public" | "authenticated" | "admin";

export interface RouteSpec {
  handler: Handler;
  access?: AccessLevel;
  /** Skip the audit-trail write for this route (avoids recursion on audit reads). */
  noAudit?: boolean;
}

export type Handler = (ctx: RequestContext) => Promise<ServerResponse>;

export function header(ctx: { headers: Record<string, string> }, name: string): string | undefined {
  const target = name.toLowerCase();
  for (const [key, value] of Object.entries(ctx.headers)) {
    if (key.toLowerCase() === target) return value;
  }
  return undefined;
}

export function modifier(ctx: RequestContext, name: string): string | undefined {
  const raw = ctx.modifiers?.[name];
  const value = Array.isArray(raw) ? raw[0] : raw;
  return value == null ? undefined : String(value);
}

export function payloadAs<T>(ctx: RequestContext, fallback: T): T {
  return (ctx.payload ?? fallback) as T;
}

export function requirePayload(ctx: RequestContext): Record<string, unknown> {
  if (ctx.payload == null || typeof ctx.payload !== "object" || Array.isArray(ctx.payload)) {
    throw new Error("expected a JSON object payload");
  }
  return ctx.payload as Record<string, unknown>;
}
