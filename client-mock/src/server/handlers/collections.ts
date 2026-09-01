/**
 * Collections + document handlers.
 *
 * Covers collection metadata CRUD, the generic document query engine
 * (`system:collections:document:query`) used by collections, users pagers,
 * policies, and the audit-log query endpoint.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { executeQuery } from "../../query";
import { clone } from "../../util";
import { requirePayload, payloadAs } from "../context";
import {
  applyUpdate,
  AUDIT_LOG_COLLECTION,
  deleteCollectionData,
  deleteDoc,
  ensureCollection,
  getDoc,
  listDocs,
  newDoc,
  putDoc,
  requireDoc,
  sanitizeUser,
  stripInternal,
  USER_COLLECTION,
} from "../store";

interface QueryPayload {
  pagination?: { type?: string; offset?: number; limit?: number };
}

export function collectionHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const queryCollection = (collection: string, sanitize?: (doc: Record<string, unknown>) => Record<string, unknown>) =>
    async (ctx: Parameters<RouteSpec["handler"]>[0]) => {
      const payload = payloadAs<QueryPayload>(ctx, {});
      let docs = await listDocs(tables, collection);
      if (sanitize) docs = docs.map(sanitize) as typeof docs;
      const offset = Math.max(0, payload.pagination?.offset ?? 0);
      const limit = Math.max(1, payload.pagination?.limit ?? docs.length);
      const { items, total } = executeQuery(
        docs as unknown as Record<string, unknown>[],
        ctx.payload as never,
      );
      return ok(
        items,
        {
          page: {
            number: Math.floor(offset / limit) + 1,
            size: limit,
            count: items.length,
            total,
            pages: Math.max(1, Math.ceil(total / limit)),
          },
        },
      );
    };

  return {
    "system:collections:collection:list": {
      access: "authenticated",
      handler: async () => {
        const items = await tables.collections.getAll();
        return ok(items);
      },
    },

    "system:collections:collection:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const meta = await tables.collections.get(ctx.args["name"]);
        if (!meta) throw err.notFound(`collection "${ctx.args["name"]}"`);
        return ok(meta);
      },
    },

    "system:collections:collection:create": {
      access: "authenticated",
      handler: async (ctx) => {
        const schema = requirePayload(ctx) as Record<string, unknown>;
        const name = String(schema["name"] ?? "");
        if (!name) throw err.required("schema.name");
        if (await tables.collections.get(name)) {
          throw err.duplicate(`collection "${name}"`);
        }
        await tables.collections.put(schema);
        return ok(schema);
      },
    },

    "system:collections:collection:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        const meta = await tables.collections.get(name);
        if (!meta) throw err.notFound(`collection "${name}"`);
        await deleteCollectionData(tables, name);
        return noContent();
      },
    },

    "system:collections:document:query": {
      access: "authenticated",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        if (!(await tables.collections.get(name))) {
          throw err.notFound(`collection "${name}"`);
        }
        const docs = await listDocs(tables, name);
        const sanitized =
          name === USER_COLLECTION ? docs.map(sanitizeUser) : docs;
        const payload = payloadAs<QueryPayload>(ctx, {});
        const offset = Math.max(0, payload.pagination?.offset ?? 0);
        const limit = Math.max(1, payload.pagination?.limit ?? sanitized.length);
        const { items, total } = executeQuery(
          sanitized as unknown as Record<string, unknown>[],
          ctx.payload as never,
        );
        return ok(
          items,
          {
            page: {
              number: Math.floor(offset / limit) + 1,
              size: limit,
              count: items.length,
              total,
              pages: Math.max(1, Math.ceil(total / limit)),
            },
          },
        );
      },
    },

    "system:collections:document:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const { name, doc_id } = ctx.args;
        const doc = await getDoc(tables, name, doc_id);
        if (!doc) throw err.notFound(`document "${doc_id}" in collection "${name}"`);
        const view = stripInternal(clone(doc));
        return ok(name === USER_COLLECTION ? sanitizeUser(view) : view);
      },
    },

    "system:collections:document:create": {
      access: "authenticated",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        const payload = payloadAs<Record<string, unknown>>(ctx, {});
        await ensureCollection(tables, name);
        const doc = await newDoc(name, clone(payload));
        await putDoc(tables, doc);
        deps.bus.emit(`collection:${name}`, { event: "created", document: stripInternal(clone(doc)) });
        return ok(stripInternal(clone(doc)));
      },
    },

    "system:collections:document:update": {
      access: "authenticated",
      handler: async (ctx) => {
        const { name, doc_id } = ctx.args;
        const current = await requireDoc(tables, name, doc_id);
        const payload = payloadAs<Record<string, unknown>>(ctx, {});
        const updated = await applyUpdate(current, clone(payload));
        await putDoc(tables, updated);
        deps.bus.emit(`collection:${name}`, { event: "updated", document: stripInternal(clone(updated)) });
        return ok(stripInternal(clone(updated)));
      },
    },

    "system:collections:document:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        const { name, doc_id } = ctx.args;
        await deleteDoc(tables, name, doc_id);
        deps.bus.emit(`collection:${name}`, { event: "deleted", document: { _id_: doc_id } });
        return noContent();
      },
    },

    // Audit-log query shares the document query engine over `_audit_log_`.
    "system:collections:audit_log:query": {
      access: "authenticated",
      noAudit: true,
      handler: queryCollection(AUDIT_LOG_COLLECTION),
    },
  };
}
