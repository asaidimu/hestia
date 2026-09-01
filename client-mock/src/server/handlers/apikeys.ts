/**
 * API key handlers: create (returns the only-time plaintext secret),
 * list/get/update/rotate/delete.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { clone, randomHex } from "../../util";
import { requirePayload } from "../context";
import { API_KEY_COLLECTION, applyUpdate, newDoc, putDoc, requireDoc } from "../store";

export function apiKeyHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const publicKey = (doc: Record<string, unknown>) => {
    // `key` (the plaintext secret) is only ever returned on create/rotate.
    const { key: _ignored, ...rest } = doc;
    return rest;
  };

  return {
    "system:apikeys:key:create": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const name = String(payload["name"] ?? "");
        if (!name) throw err.required("name");

        const secret = `hst_${randomHex(20)}`;
        const doc = await newDoc(API_KEY_COLLECTION, {
          name,
          prefix: secret.slice(0, 10),
          key: secret,
          operations: Array.isArray(payload["operations"]) ? payload["operations"] : ["*"],
          status: "active",
          environment: payload["environment"] ?? "development",
          expiry: payload["expiry"] ?? null,
          usage: 0,
          last_used: null,
          user_id: ctx.identity!.user._id_,
          tenant_id: ctx.identity!.user["tenant_id"] ?? "root",
        });
        await putDoc(tables, doc);
        return ok(clone(doc));
      },
    },

    "system:apikeys:key:list": {
      access: "authenticated",
      handler: async (ctx) => {
        const keys = await tables.documents.getAllByIndex("by_collection", API_KEY_COLLECTION);
        const own = keys.filter((k) => k["user_id"] === ctx.identity!.user._id_ || ctx.identity!.is_admin);
        return ok(own.map(publicKey));
      },
    },

    "system:apikeys:key:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const doc = await requireDoc(tables, API_KEY_COLLECTION, ctx.args["key_id"]);
        return ok(publicKey(clone(doc)));
      },
    },

    "system:apikeys:key:update": {
      access: "authenticated",
      handler: async (ctx) => {
        const current = await requireDoc(tables, API_KEY_COLLECTION, ctx.args["key_id"]);
        const payload = requirePayload(ctx);
        const allowed: Record<string, unknown> = {};
        for (const field of ["name", "environment", "operations", "status", "expiry"] as const) {
          if (payload[field] !== undefined) allowed[field] = payload[field];
        }
        const updated = await applyUpdate(current, allowed);
        await putDoc(tables, updated);
        return ok(publicKey(clone(updated)));
      },
    },

    "system:apikeys:key:rotate": {
      access: "authenticated",
      handler: async (ctx) => {
        const current = await requireDoc(tables, API_KEY_COLLECTION, ctx.args["key_id"]);
        const secret = `hst_${randomHex(20)}`;
        const updated = await applyUpdate(current, {
          key: secret,
          prefix: secret.slice(0, 10),
        });
        await putDoc(tables, updated);
        return ok(clone(updated));
      },
    },

    "system:apikeys:key:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        await requireDoc(tables, API_KEY_COLLECTION, ctx.args["key_id"]);
        await tables.documents.delete([API_KEY_COLLECTION, ctx.args["key_id"]]);
        return noContent();
      },
    },
  };
}
