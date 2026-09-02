/**
 * Identity resolution, session management, and route access control.
 *
 * Sessions are persisted in IndexedDB (alongside everything else), so the
 * "server" remembers who is logged in across page reloads — matching the
 * offline-first semantics of the mock.
 */

import type { MockTables, SessionRecord, StoredDocument } from "../schema";
import { err } from "../errors";
import { hashPassword, randomToken, verifyPassword } from "../util";
import { header, type AccessLevel, type RequestContext, type ResolvedIdentity } from "./context";
import { API_KEY_COLLECTION, USER_COLLECTION } from "./store";

export const CURRENT_SESSION_KEY = "current_session";
export const SESSION_TTL_MS = 7 * 24 * 60 * 60 * 1000;

/** Route names reachable without any credentials. */
const PUBLIC_ROUTES = new Set<string>([
  "system:core:health:check",
  "system:auth:session:create",
  "system:auth:password:reset",
  "system:auth:password:confirm",
  "system:users:user:create", // register
]);

/** Route names restricted to administrators. */
const ADMIN_ROUTES = new Set<string>([
  "system:users:user:delete",
  "system:notifications:notification:create",
  "system:schedules:schedule:all",
  "system:policies:policy:create",
  "system:policies:policy:update",
  "system:policies:rule:create",
  "system:policies:rule:update",
  "system:policies:rule:delete",
  "system:policies:rule:validate",
  "system:policies:reload",
  "system:policies:binding:get",
  "system:policies:binding:list",
  "system:audit:log:export",
  "system:audit:log:stream",
  "system:logs:list",
  "system:logs:stream",
  "system:updates:status:get",
  "system:updates:changelog:get",
  "system:updates:check:create",
  "system:updates:check:get",
  "system:updates:stage:create",
  "system:updates:update:apply",
  "system:updates:update:discard",
  "system:core:reset",
  "system:core:capability:set",
]);

export function accessForRoute(route: string): AccessLevel {
  if (PUBLIC_ROUTES.has(route)) return "public";
  if (ADMIN_ROUTES.has(route)) return "admin";
  return "authenticated";
}

async function userById(tables: MockTables, id: string): Promise<StoredDocument | undefined> {
  return await tables.documents.get([USER_COLLECTION, id]);
}

async function resolveApiKeyIdentity(
  tables: MockTables,
  apiKey: string,
): Promise<ResolvedIdentity | null> {
  const keys = await tables.documents.getAllByIndex("by_collection", API_KEY_COLLECTION);
  const match = keys.find((k) => k["key"] === apiKey && k["status"] === "active");
  if (!match) return null;
  const user = await userById(tables, String(match["user_id"] ?? ""));
  if (!user) return null;
  return {
    user,
    auth_method: "api_key",
    api_key_id: match._id_,
    is_admin: isAdmin(user),
  };
}

export function isAdmin(user: StoredDocument): boolean {
  const permissions = user["permissions"];
  return Array.isArray(permissions) && permissions.includes("administrator");
}

/**
 * Resolve the caller identity from, in order of precedence:
 *  1. `X-API-Key` header (active API key owned by a user)
 *  2. the persisted current session token
 */
export async function resolveIdentity(
  tables: MockTables,
  ctx: { headers: Record<string, string> },
): Promise<ResolvedIdentity | null> {
  const apiKey = header(ctx, "X-API-Key");
  if (apiKey) {
    const identity = await resolveApiKeyIdentity(tables, apiKey);
    if (identity) return { ...identity, auth_method: "bootstrap_key" };
    return null;
  }

  const pointer = (await tables.kv.get(CURRENT_SESSION_KEY)) as { v?: string } | undefined;
  const token = pointer?.v;
  if (!token) return null;

  const session = await tables.sessions.get(token);
  if (!session) return null;
  if (session.expires_at < Date.now()) {
    await tables.sessions.delete(token);
    if ((await tables.kv.get(CURRENT_SESSION_KEY)) === pointer) {
      await tables.kv.delete(CURRENT_SESSION_KEY);
    }
    return null;
  }

  const user = await userById(tables, session.user_id);
  if (!user) return null;
  return {
    user,
    auth_method: "password",
    session_id: session.token,
    is_admin: isAdmin(user),
  };
}

export function checkAccess(access: AccessLevel, identity: ResolvedIdentity | null): void {
  if (access === "public") return;
  if (!identity) throw err.unauthenticated();
  if (access === "admin" && !identity.is_admin) {
    throw err.denied("this administrator-only operation");
  }
}

export async function createSession(tables: MockTables, userId: string): Promise<SessionRecord> {
  const now = Date.now();
  const record: SessionRecord = {
    token: randomToken("hst_sess"),
    user_id: userId,
    created_at: now,
    expires_at: now + SESSION_TTL_MS,
  };
  await tables.sessions.put(record);
  await tables.kv.put({ k: CURRENT_SESSION_KEY, v: record.token });
  return record;
}

export async function revokeCurrentSession(tables: MockTables): Promise<void> {
  const pointer = (await tables.kv.get(CURRENT_SESSION_KEY)) as { v?: string } | undefined;
  const token = pointer?.v;
  if (!token) throw err.noActiveSession();
  const session = await tables.sessions.get(token);
  if (!session) {
    await tables.kv.delete(CURRENT_SESSION_KEY);
    throw err.noActiveSession();
  }
  await tables.sessions.delete(token);
  await tables.kv.delete(CURRENT_SESSION_KEY);
}

export async function setCredentials(
  tables: MockTables,
  userId: string,
  password: string,
): Promise<void> {
  const user = await userById(tables, userId);
  if (!user) throw err.notFound("user");
  user["password"] = await hashPassword(password);
  await tables.documents.put(user);
}

export async function verifyCredentials(
  tables: MockTables,
  email: string,
  password: string,
): Promise<StoredDocument | null> {
  const users = await tables.documents.getAllByIndex("by_collection", USER_COLLECTION);
  const user = users.find((u) => u["email"] === email && u["deleted"] == null);
  if (!user) return null;
  const ok = await verifyPassword(password, String(user["password"] ?? ""));
  return ok ? user : null;
}
