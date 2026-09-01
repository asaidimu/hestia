/**
 * Schedule (cron) handlers.
 *
 * Schedules are stored documents. The mock does not run a cron daemon; it
 * supports `@every <n><unit>` expressions for auto-ticking (when
 * `options.autoTickSchedules` is enabled) and exposes `server.fireSchedule()`
 * for manual/programmatic dispatching.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { clone } from "../../util";
import { requirePayload } from "../context";
import { applyUpdate, newDoc, putDoc, requireDoc } from "../store";

export const SCHEDULES_COLLECTION = "_scheduled_messages_";

export function scheduleHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const collection = () => SCHEDULES_COLLECTION;

  return {
    "system:schedules:schedule:create": {
      access: "authenticated",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const message = String(payload["message"] ?? "");
        const cron = String(payload["cron"] ?? "");
        if (!message) throw err.required("message");
        if (!cron) throw err.required("cron");

        const doc = await newDoc(collection(), {
          user_id: String(payload["user_id"] ?? ctx.identity!.user._id_),
          message,
          input: payload["input"] ?? {},
          cron,
          disabled: payload["disabled"] === true,
          tenant_id: ctx.identity!.user["tenant_id"] ?? "root",
          created_at: Date.now(),
        });
        await putDoc(tables, doc);
        return ok({ id: doc._id_ });
      },
    },

    "system:schedules:schedule:list": {
      access: "authenticated",
      handler: async (ctx) => {
        const all = await tables.documents.getAllByIndex("by_collection", collection());
        const own = all.filter((s) => s["user_id"] === ctx.identity!.user._id_ || ctx.identity!.is_admin);
        return ok(own.map(clone));
      },
    },

    "system:schedules:schedule:all": {
      access: "admin",
      handler: async () => {
        const all = await tables.documents.getAllByIndex("by_collection", collection());
        return ok(all.map(clone));
      },
    },

    "system:schedules:schedule:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const doc = await requireDoc(tables, collection(), ctx.args["id"]);
        return ok(clone(doc));
      },
    },

    "system:schedules:schedule:update": {
      access: "authenticated",
      handler: async (ctx) => {
        const current = await requireDoc(tables, collection(), ctx.args["id"]);
        const payload = requirePayload(ctx);
        const patch: Record<string, unknown> = {};
        for (const field of ["message", "input", "cron", "disabled"] as const) {
          if (payload[field] !== undefined) patch[field] = payload[field];
        }
        const updated = await applyUpdate(current, patch);
        await putDoc(tables, updated);
        return noContent();
      },
    },

    "system:schedules:schedule:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        await requireDoc(tables, collection(), ctx.args["id"]);
        await tables.documents.delete([collection(), ctx.args["id"]]);
        return noContent();
      },
    },
  };
}
