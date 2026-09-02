/**
 * Auth + user management handlers.
 *
 * Response shapes mirror the Go `AuthService`:
 *  - session:create → `{data: {user: <sanitized user doc>}}` + new session
 *  - users:user:create → `{data: <user doc>}` (public registration)
 *  - reads never include the password hash
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { executeQuery } from "../../query";
import { clone, hashPassword, nowIso, randomToken, verifyPassword } from "../../util";
import { requirePayload } from "../context";
import {
  createSession,
  isAdmin,
  revokeCurrentSession,
  setCredentials,
  verifyCredentials,
} from "../auth";
import {
  applyUpdate,
  newDoc,
  putDoc,
  requireDoc,
  sanitizeUser,
  USER_COLLECTION,
} from "../store";

export function userHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  return {
    "system:auth:session:create": {
      access: "public",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const email = String(payload["email"] ?? "");
        const password = String(payload["password"] ?? "");
        if (!email || !password) throw err.invalidCredentials();

        const user = await verifyCredentials(tables, email, password);
        if (!user) throw err.invalidCredentials();

        const session = await createSession(tables, user._id_);

        // Attribute the audit entry to the newly authenticated identity.
        ctx.identity = {
          user,
          auth_method: "password",
          session_id: session.token,
          is_admin: isAdmin(user),
        };

        return ok({ user: sanitizeUser(clone(user)) });
      },
    },

    "system:auth:session:delete": {
      access: "authenticated",
      handler: async () => {
        await revokeCurrentSession(tables);
        return noContent();
      },
    },

    "system:auth:token:elevate": {
      access: "authenticated",
      handler: async (ctx) => {
        // The real server issues a short-lived elevated token; the mock keeps
        // the current session and reports elevation via a fresh timestamp.
        return ok({ token: ctx.identity!.session_id, elevated: true, ts: nowIso() });
      },
    },

    "system:auth:password:reset": {
      access: "public",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const email = String(payload["email"] ?? "");
        if (!email) throw err.required("email");

        const token = randomToken("hst_reset");
        await tables.password_resets.put({
          token,
          email,
          created_at: Date.now(),
          expires_at: Date.now() + 60 * 60 * 1000,
        });
        // Unlike the real server (which emails the token), the mock returns
        // it so the flow is completable in tests and local development.
        return ok({ email, token });
      },
    },

    "system:auth:password:confirm": {
      access: "public",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const token = String(payload["token"] ?? "");
        const password = String(payload["password"] ?? "");
        if (!token) throw err.required("token");
        if (!password) throw err.required("password");

        const record = await tables.password_resets.get(token);
        if (!record || record.expires_at < Date.now()) {
          throw err.validation("reset token is invalid or expired");
        }

        const users = await tables.documents.getAllByIndex("by_collection", USER_COLLECTION);
        const user = users.find((u) => u["email"] === record.email);
        if (!user) throw err.notFound("user");

        await setCredentials(tables, user._id_, password);
        await tables.password_resets.delete(token);
        return noContent();
      },
    },

    "system:auth:bootstrap:password:set": {
      access: "public",
      handler: async (ctx) => {
        // Requires a valid X-API-Key (resolved as identity with auth_method bootstrap_key).
        if (!ctx.identity || ctx.identity.auth_method !== "bootstrap_key") {
          throw err.unauthenticated();
        }
        const payload = requirePayload(ctx);
        const password = String(payload["password"] ?? "");
        const email = String(payload["email"] ?? "");
        if (!password) throw err.required("password");

        const user = ctx.identity.user;
        if (email) user["email"] = email;
        await setCredentials(tables, user._id_, password);
        await deps.setBootstrapped(true);
        return noContent();
      },
    },

    "system:users:user:create": {
      access: "public",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const email = String(payload["email"] ?? "");
        const password = String(payload["password"] ?? "");
        const name = String(payload["name"] ?? "");
        if (!email || !password) throw err.validation("email and password are required");

        const users = await tables.documents.getAllByIndex("by_collection", USER_COLLECTION);
        if (users.some((u) => u["email"] === email)) {
          throw err.duplicate(`user with email "${email}"`);
        }

        const doc = await newDoc(USER_COLLECTION, {
          email,
          name,
          password: await hashPassword(password),
          verified: payload["verified"] === true,
          permissions: Array.isArray(payload["permissions"]) ? payload["permissions"] : ["authenticated"],
          tenant_id: String(payload["tenant_id"] ?? "root"),
          deleted: null,
        });
        await putDoc(tables, doc);
        return ok(sanitizeUser(clone(doc)));
      },
    },

    "system:users:user:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const doc = await requireDoc(tables, USER_COLLECTION, ctx.args["user_id"]);
        return ok(sanitizeUser(clone(doc)));
      },
    },

    "system:users:user:update": {
      access: "authenticated",
      handler: async (ctx) => {
        const current = await requireDoc(tables, USER_COLLECTION, ctx.args["user_id"]);
        const payload = requirePayload(ctx);

        // Only administrators may modify another user's record.
        if (!ctx.identity!.is_admin && current._id_ !== ctx.identity!.user._id_) {
          throw err.denied("updating another user");
        }
        if (payload["password"] !== undefined) {
          throw err.denied("changing a password through user:update");
        }

        const updated = await applyUpdate(current, payload);
        await putDoc(tables, updated);
        return ok(sanitizeUser(clone(updated)));
      },
    },

    "system:users:user:delete": {
      access: "admin",
      handler: async (ctx) => {
        const current = await requireDoc(tables, USER_COLLECTION, ctx.args["user_id"]);
        // Soft-delete, mirroring the real server's tombstone behaviour.
        const updated = await applyUpdate(current, { deleted: nowIso() });
        await putDoc(tables, updated);

        // Tear down the deleted user's sessions.
        const sessions = await tables.sessions.getAllByIndex("by_user", current._id_);
        for (const session of sessions) {
          await tables.sessions.delete(session.token);
        }
        return noContent();
      },
    },

    "system:users:password:change": {
      access: "authenticated",
      handler: async (ctx) => {
        const targetId = ctx.args["user_id"];
        const isSelf = ctx.identity!.user._id_ === targetId;
        if (!isSelf && !ctx.identity!.is_admin) {
          throw err.denied("changing another user's password");
        }

        const payload = requirePayload(ctx);
        const current = String(payload["current"] ?? "");
        const next = String(payload["new"] ?? "");

        const user = await requireDoc(tables, USER_COLLECTION, targetId);
        if (isSelf && !(await verifyPassword(current, String(user["password"] ?? "")))) {
          throw err.invalidCredentials();
        }
        await setCredentials(tables, targetId, next);

        // Password change invalidates the user's other sessions.
        const sessions = await tables.sessions.getAllByIndex("by_user", targetId);
        for (const session of sessions) {
          if (ctx.identity!.session_id && session.token !== ctx.identity!.session_id) {
            await tables.sessions.delete(session.token);
          }
        }
        return noContent();
      },
    },

    "system:collections:user:query": {
      access: "authenticated",
      handler: async (ctx) => {
        const all = await tables.documents.getAllByIndex("by_collection", USER_COLLECTION);
        const live = all.filter((u) => u["deleted"] == null).map(sanitizeUser);
        const offset = payloadOffset(ctx.payload);
        const limit = payloadLimit(ctx.payload);
        const { items, total } = executeQuery(live as Record<string, unknown>[], ctx.payload as never);
        return ok(items, { page: page(total, offset, limit, items.length) });
      },
    },
  };
}

function payloadOffset(payload: unknown): number {
  const p = payload as { pagination?: { offset?: number } } | undefined;
  return Math.max(0, p?.pagination?.offset ?? 0);
}

function payloadLimit(payload: unknown): number {
  const p = payload as { pagination?: { limit?: number } } | undefined;
  return Math.max(1, p?.pagination?.limit ?? 50);
}

function page(total: number, offset: number, limit: number, count: number) {
  const size = Math.max(1, limit);
  return {
    number: Math.floor(offset / size) + 1,
    size,
    count,
    total,
    pages: Math.max(1, Math.ceil(total / size)),
  };
}
