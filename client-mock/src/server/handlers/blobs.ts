/**
 * Blob storage handlers: namespaces, direct upload, resumable/staged upload
 * (begin → chunk → complete), download, metadata, stats, verify, compact.
 *
 * Binary payloads arrive as Blob/File/ArrayBuffer inside `dispatch` payloads;
 * they are converted to `Uint8Array` and persisted in IndexedDB, which can
 * store them natively.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { clone, isBinary, nowIso, randomToken, toBytes } from "../../util";
import { header, modifier, requirePayload } from "../context";
import type { BlobRecord, NamespaceRecord, UploadSessionRecord } from "../../schema";

function publicMeta(record: BlobRecord) {
  return { ...record.meta };
}

export function blobHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const requireNamespace = async (ns: string): Promise<NamespaceRecord> => {
    const record = await tables.namespaces.get(ns);
    if (!record) throw err.blobNotFound(`namespace "${ns}"`);
    return record;
  };

  const parseSession = async (ctx: Parameters<RouteSpec["handler"]>[0]) => {
    const sessionId =
      header(ctx, "X-Session-ID") ?? modifier(ctx, "session_id") ?? "";
    if (!sessionId) throw err.required("X-Session-ID");
    const session = await tables.upload_sessions.get(sessionId);
    if (!session) throw err.blobNotFound(`upload session "${sessionId}"`);
    return session;
  };

  return {
    "system:blobs:namespace:create": {
      access: "authenticated",
      handler: async (ctx) => {
        const ns = ctx.args["ns"];
        if (!ns) throw err.required("ns");
        if (await tables.namespaces.get(ns)) throw err.duplicate(`namespace "${ns}"`);
        const payload = (ctx.payload ?? {}) as Record<string, unknown>;
        const record: NamespaceRecord = {
          ns,
          id: ns,
          display_name: String(payload["display_name"] ?? ns),
          custom: (payload["custom"] as Record<string, string>) ?? {},
          created_at: nowIso(),
        };
        await tables.namespaces.put(record);
        return ok(clone(record));
      },
    },

    "system:blobs:namespace:list": {
      access: "authenticated",
      handler: async () => {
        const records = await tables.namespaces.getAll();
        return ok({
          namespaces: records.map((r) => ({ id: r.id, display_name: r.display_name, custom: r.custom })),
        });
      },
    },

    "system:blobs:namespace:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        const ns = ctx.args["ns"];
        await requireNamespace(ns);
        await tables.namespaces.delete(ns);
        const blobs = await tables.blobs.getAllByIndex("by_ns", ns);
        for (const blob of blobs) {
          await tables.blobs.delete([ns, blob.key]);
        }
        return noContent();
      },
    },

    "system:blobs:namespace:stats": {
      access: "authenticated",
      handler: async (ctx) => {
        const ns = ctx.args["ns"];
        await requireNamespace(ns);
        const blobs = await tables.blobs.getAllByIndex("by_ns", ns);
        const bytes = blobs.reduce((sum, b) => sum + (b.meta.size ?? 0), 0);
        return ok({
          namespace_id: ns,
          blob_count: blobs.length,
          bytes_stored: bytes,
          bytes_physical: bytes,
          chunk_count: blobs.length,
          dead_bytes: 0,
          dead_chunks: 0,
          segment_count: Math.max(1, blobs.length),
        });
      },
    },

    "system:blobs:namespace:verify": {
      access: "authenticated",
      handler: async (ctx) => {
        await requireNamespace(ctx.args["ns"]);
        return noContent();
      },
    },

    "system:blobs:namespace:compact": {
      access: "authenticated",
      handler: async (ctx) => {
        await requireNamespace(ctx.args["ns"]);
        return ok({ blobs_removed: 0, chunks_removed: 0, bytes_freed: 0, segments_compacted: 0 });
      },
    },

    "system:blobs:blob:upload": {
      access: "authenticated",
      handler: async (ctx) => {
        const { ns, key } = ctx.args;
        await requireNamespace(ns);
        if (!isBinary(ctx.payload)) throw err.validation("expected a binary upload body");

        const overwrite = modifier(ctx, "overwrite") === "true";
        const existing = await tables.blobs.get([ns, key]);
        if (existing && !overwrite) throw err.duplicate(`blob "${ns}/${key}"`);

        const data = await toBytes(ctx.payload);
        const contentType = header(ctx, "Content-Type") ?? "application/octet-stream";
        const now = nowIso();
        const record: BlobRecord = {
          ns,
          key,
          meta: {
            key,
            namespace_id: ns,
            content_type: contentType,
            size: data.byteLength,
            created_at: existing?.meta.created_at ?? now,
            updated_at: existing ? now : undefined,
          },
          data,
        };
        await tables.blobs.put(record);
        return ok(publicMeta(record));
      },
    },

    "system:blobs:blob:begin": {
      access: "authenticated",
      handler: async (ctx) => {
        const ns = ctx.args["ns"];
        await requireNamespace(ns);
        const payload = requirePayload(ctx);
        const key = String(payload["key"] ?? "");
        if (!key) throw err.required("key");
        const size = Number(payload["size"] ?? 0);
        if (!Number.isFinite(size) || size <= 0) throw err.validation("size must be a positive number");

        const existing = await tables.blobs.get([ns, key]);
        const overwrite = modifier(ctx, "overwrite") === "true";
        if (existing && !overwrite) throw err.duplicate(`blob "${ns}/${key}"`);

        const session: UploadSessionRecord = {
          session_id: randomToken("hst_upl"),
          ns,
          key,
          expected_size: size,
          content_type: String(payload["content_type"] ?? "application/octet-stream"),
          block_size: Number(payload["block_size"] ?? 8 * 1024 * 1024),
          overwrite,
          chunks: [],
          received: 0,
          created_at: Date.now(),
        };
        await tables.upload_sessions.put(session);
        return ok({
          session_id: session.session_id,
          key,
          offset: 0,
          block_size: session.block_size,
        });
      },
    },

    "system:blobs:blob:chunk": {
      access: "authenticated",
      handler: async (ctx) => {
        const ns = ctx.args["ns"];
        const session = await parseSession(ctx);
        if (session.ns !== ns) throw err.validation("upload session namespace mismatch");
        if (!isBinary(ctx.payload)) throw err.validation("expected a binary chunk body");

        const offset = Number(header(ctx, "X-Offset") ?? "0");
        if (!Number.isFinite(offset) || offset < 0) throw err.validation("X-Offset must be a non-negative number");

        const data = await toBytes(ctx.payload);
        session.chunks.push({ offset, data });
        session.received += data.byteLength;
        await tables.upload_sessions.put(session);
        return ok({ total: session.received });
      },
    },

    "system:blobs:blob:progress": {
      access: "authenticated",
      handler: async (ctx) => {
        const session = await parseSession(ctx);
        const ranges = session.chunks
          .sort((a, b) => a.offset - b.offset)
          .map((c) => ({ start: c.offset, end: c.offset + c.data.byteLength }));
        return ok({
          total: session.received,
          ranges,
          block_size: session.block_size,
          expected_size: session.expected_size,
        });
      },
    },

    "system:blobs:blob:complete": {
      access: "authenticated",
      handler: async (ctx) => {
        const ns = ctx.args["ns"];
        const session = await parseSession(ctx);
        if (session.ns !== ns) throw err.validation("upload session namespace mismatch");

        session.chunks.sort((a, b) => a.offset - b.offset);
        const totalLength = session.chunks.reduce((max, c) => Math.max(max, c.offset + c.data.byteLength), 0);
        const assembled = new Uint8Array(totalLength);
        for (const chunk of session.chunks) {
          assembled.set(chunk.data, chunk.offset);
        }

        const now = nowIso();
        const existing = await tables.blobs.get([ns, session.key]);
        const record: BlobRecord = {
          ns,
          key: session.key,
          meta: {
            key: session.key,
            namespace_id: ns,
            content_type: session.content_type,
            size: assembled.byteLength,
            created_at: existing?.meta.created_at ?? now,
            updated_at: existing ? now : undefined,
          },
          data: assembled,
        };
        await tables.blobs.put(record);
        await tables.upload_sessions.delete(session.session_id);
        return ok(publicMeta(record));
      },
    },

    "system:blobs:blob:abort": {
      access: "authenticated",
      handler: async (ctx) => {
        const session = await parseSession(ctx);
        await tables.upload_sessions.delete(session.session_id);
        return noContent();
      },
    },

    "system:blobs:blob:head": {
      access: "authenticated",
      handler: async (ctx) => {
        const { ns, key } = ctx.args;
        await requireNamespace(ns);
        const record = await tables.blobs.get([ns, key]);
        if (!record) throw err.blobNotFound(`blob "${ns}/${key}"`);
        return ok(publicMeta(record));
      },
    },

    "system:blobs:blob:download": {
      access: "authenticated",
      handler: async (ctx) => {
        const { ns, key } = ctx.args;
        await requireNamespace(ns);
        const record = await tables.blobs.get([ns, key]);
        if (!record) throw err.blobNotFound(`blob "${ns}/${key}"`);
        // The transport converts the byte payload into a real Blob when the
        // client requested responseType "blob".
        const body = new Uint8Array(record.data);
        return {
          status: 200,
          body: {
            data: body,
            metadata: { timestamp: nowIso(), request: ctx.request_id, content_type: record.meta.content_type },
          },
        };
      },
    },

    "system:blobs:blob:list": {
      access: "authenticated",
      handler: async (ctx) => {
        const ns = ctx.args["ns"];
        await requireNamespace(ns);
        const payload = (ctx.payload ?? {}) as { prefix?: string; limit?: number };
        let blobs = await tables.blobs.getAllByIndex("by_ns", ns);
        if (payload.prefix) blobs = blobs.filter((b) => b.key.startsWith(payload.prefix!));
        if (payload.limit) blobs = blobs.slice(0, payload.limit);
        return ok({ blobs: blobs.map(publicMeta) });
      },
    },

    "system:blobs:blob:rename": {
      access: "authenticated",
      handler: async (ctx) => {
        const { ns, key } = ctx.args;
        const payload = requirePayload(ctx);
        const newKey = String(payload["new_key"] ?? "");
        if (!newKey) throw err.required("new_key");

        const record = await tables.blobs.get([ns, key]);
        if (!record) throw err.blobNotFound(`blob "${ns}/${key}"`);
        if (await tables.blobs.get([ns, newKey])) throw err.duplicate(`blob "${ns}/${newKey}"`);

        await tables.blobs.delete([ns, key]);
        await tables.blobs.put({
          ...record,
          key: newKey,
          meta: { ...record.meta, key: newKey, updated_at: nowIso() },
        });
        return noContent();
      },
    },

    "system:blobs:blob:update": {
      access: "authenticated",
      handler: async (ctx) => {
        const { ns, key } = ctx.args;
        const payload = requirePayload(ctx);
        const record = await tables.blobs.get([ns, key]);
        if (!record) throw err.blobNotFound(`blob "${ns}/${key}"`);
        if (payload["custom"] !== undefined) {
          record.meta.custom = payload["custom"] as Record<string, unknown>;
          record.meta.updated_at = nowIso();
          await tables.blobs.put(record);
        }
        return ok(publicMeta(record));
      },
    },

    "system:blobs:blob:delete": {
      access: "authenticated",
      handler: async (ctx) => {
        const { ns, key } = ctx.args;
        const record = await tables.blobs.get([ns, key]);
        if (!record) throw err.blobNotFound(`blob "${ns}/${key}"`);
        await tables.blobs.delete([ns, key]);
        return noContent();
      },
    },
  };
}
