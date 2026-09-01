/**
 * In-app notification handlers + settings handlers.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { clone, uuid } from "../../util";
import { requirePayload } from "../context";
import { Topics } from "../../bus";
import { newDoc } from "../store";

export function notificationHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const visible = async (ctx: Parameters<RouteSpec["handler"]>[0]) => {
    const all = await tables.notifications.getAll();
    const userId = ctx.identity!.user._id_;
    return ctx.identity!.is_admin ? all : all.filter((n) => n["user_id"] === userId);
  };

  const publish = (doc: Record<string, unknown>) => {
    const userId = String(doc["user_id"] ?? "");
    const payload = { data: doc };
    deps.bus.emit(Topics.notifications(userId), payload);
    deps.bus.emit(Topics.notificationsAll, payload);
  };

  return {
    "system:notifications:notification:create": {
      access: "admin",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const userId = String(payload["user_id"] ?? "");
        const subject = String(payload["subject"] ?? "");
        if (!userId || !subject) throw err.validation("user_id and subject are required");

        const doc = await newDoc("_notifications_", {
          _id_: uuid(),
          user_id: userId,
          type: payload["type"] ?? "info",
          subject,
          body: payload["body"] ?? "",
          data: payload["data"] ?? {},
          actions: payload["actions"] ?? [],
          read: false,
          created_at: Date.now(),
          expires_at: payload["expires_at"] ?? null,
        });
        await tables.notifications.put(doc);
        publish(doc);
        return ok(clone(doc));
      },
    },

    "system:notifications:notification:list": {
      access: "authenticated",
      handler: async (ctx) => {
        const items = await visible(ctx);
        return ok(items);
      },
    },

    "system:notifications:notification:read": {
      access: "authenticated",
      handler: async (ctx) => {
        const id = ctx.args["notification_id"];
        const doc = await tables.notifications.get(id);
        if (!doc) throw err.notFound(`notification "${id}"`);
        doc["read"] = true;
        await tables.notifications.put(doc);
        return noContent();
      },
    },

    "system:notifications:read:all": {
      access: "authenticated",
      handler: async (ctx) => {
        const items = await visible(ctx);
        for (const doc of items) {
          if (!doc["read"]) {
            doc["read"] = true;
            await tables.notifications.put(doc);
          }
        }
        return noContent();
      },
    },

    "system:notifications:unread:count": {
      access: "authenticated",
      handler: async (ctx) => {
        const items = await visible(ctx);
        return ok({ count: items.filter((n) => !n["read"]).length });
      },
    },
  };
}

export function settingHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const toDoc = (key: string, value: Record<string, unknown>) => ({
    _id_: key,
    key,
    value,
  });

  return {
    "system:settings:list": {
      access: "authenticated",
      handler: async () => {
        const rows = await tables.documents.getAllByIndex("by_collection", "_settings_");
        return ok(rows.map((r) => toDoc(String(r["key"]), (r["value"] as Record<string, unknown>) ?? {})));
      },
    },

    "system:settings:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const key = ctx.args["key"];
        const doc = await tables.documents.get(["_settings_", key]);
        if (!doc) throw err.notFound(`setting "${key}"`);
        return ok(toDoc(key, (doc["value"] as Record<string, unknown>) ?? {}));
      },
    },

    "system:settings:set": {
      access: "authenticated",
      handler: async (ctx) => {
        const key = ctx.args["key"];
        const payload = requirePayload(ctx);
        const value = (payload["value"] ?? {}) as Record<string, unknown>;

        const existing = await tables.documents.get(["_settings_", key]);
        const doc = await newDoc("_settings_", { _id_: key, key, value });
        if (existing) {
          doc._metadata_ = { ...existing._metadata_, updated: doc._metadata_.updated, version: existing._metadata_.version + 1 };
        }
        await tables.documents.put(doc);
        return ok(toDoc(key, value));
      },
    },

    "system:settings:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        const key = ctx.args["key"];
        const doc = await tables.documents.get(["_settings_", key]);
        if (!doc) throw err.notFound(`setting "${key}"`);
        await tables.documents.delete(["_settings_", key]);
        return noContent();
      },
    },
  };
}
