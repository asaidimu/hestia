/**
 * Core system handlers: health, heartbeat, capabilities, docs list, reset.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { requirePayload } from "../context";

export function coreHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  return {
    "system:core:health:check": {
      access: "public",
      handler: async () => {
        const bootstrapped = await deps.isBootstrapped();
        return ok({ ok: true, bootstrapped });
      },
    },

    "system:core:heartbeat": {
      access: "authenticated",
      handler: async () => ok({ ok: true, ts: Date.now() }),
    },

    "system:core:capability:list": {
      access: "authenticated",
      handler: async () => {
        const items = await tables.capabilities.getAll();
        return ok(items);
      },
    },

    "system:core:capability:set": {
      access: "admin",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        if (!name) throw err.required("name");
        const payload = requirePayload(ctx);
        const existing = (await tables.capabilities.get(name)) ?? {};
        const doc = { ...existing, name, ...payload };
        await tables.capabilities.put(doc);
        return ok(doc);
      },
    },

    "system:core:docs:list": {
      access: "authenticated",
      handler: async () => {
        const docs = (await tables.kv.get("core_docs")) as { v?: unknown[] } | undefined;
        return ok(docs?.v ?? []);
      },
    },

    "system:core:reset": {
      access: "admin",
      noAudit: true,
      handler: async () => {
        await deps.reset();
        return noContent();
      },
    },
  };
}
