/**
 * Document-store helpers shared by every handler.
 *
 * All documents — user data and system collections (`_user_`, `_api_key_`,
 * `_operation_policy_`, `_audit_log_`, ...) — live in the single `documents`
 * object store keyed by `[collection, _id_]`, exactly like the real server's
 * anansi-managed tables. System documents therefore work with the generic
 * `system:collections:document:*` routes (the policies pager relies on this).
 */

import type { MockTables, StoredDocument } from "../schema";
import { err } from "../errors";
import { nowIso, sha256Hex, stableStringify, uuid } from "../util";

export const USER_COLLECTION = "_user_";
export const API_KEY_COLLECTION = "_api_key_";
export const POLICY_COLLECTION = "_operation_policy_";
export const AUDIT_LOG_COLLECTION = "_audit_log_";

export function stripInternal<T extends Record<string, unknown>>(doc: T): T {
  const { collection: _ignored, ...rest } = doc;
  return rest as T;
}

/** Remove the password hash from a user document before returning it. */
export function sanitizeUser<T extends Record<string, unknown>>(doc: T): T {
  const { password: _ignored, ...rest } = doc;
  return rest as T;
}

export async function getDoc(
  tables: MockTables,
  collection: string,
  id: string,
): Promise<StoredDocument | undefined> {
  return await tables.documents.get([collection, id]);
}

export async function requireDoc(
  tables: MockTables,
  collection: string,
  id: string,
): Promise<StoredDocument> {
  const doc = await tables.documents.get([collection, id]);
  if (!doc) throw err.notFound(`document "${id}" in collection "${collection}"`);
  return doc;
}

export async function putDoc(tables: MockTables, doc: StoredDocument): Promise<void> {
  await tables.documents.put(doc);
}

export async function listDocs(tables: MockTables, collection: string): Promise<StoredDocument[]> {
  return await tables.documents.getAllByIndex("by_collection", collection);
}

export async function deleteDoc(tables: MockTables, collection: string, id: string): Promise<void> {
  const existing = await tables.documents.get([collection, id]);
  if (!existing) throw err.notFound(`document "${id}" in collection "${collection}"`);
  await tables.documents.delete([collection, id]);
}

export async function computeChecksum(data: unknown): Promise<string> {
  return await sha256Hex(stableStringify(data));
}

/**
 * Create a new stored document: assigns `_id_`, stamps `_metadata_` with a
 * checksum, timestamps, and version 1 (optimistic-locking baseline).
 */
export async function newDoc(
  collection: string,
  data: Record<string, unknown>,
): Promise<StoredDocument> {
  const { _id_, _metadata_, ...rest } = data;
  const id = typeof _id_ === "string" && _id_ ? _id_ : uuid();
  const now = nowIso();
  return {
    collection,
    _id_: id,
    _metadata_: {
      checksum: await computeChecksum(rest),
      created: now,
      updated: now,
      version: 1,
    },
    ...rest,
  };
}

/**
 * Merge a patch into an existing document with optimistic-locking semantics:
 * bumps `version`, refreshes the checksum, and rejects stale `_metadata_`
 * versions supplied by the caller.
 */
export async function applyUpdate(
  current: StoredDocument,
  patch: Record<string, unknown>,
): Promise<StoredDocument> {
  const incomingMeta = patch._metadata_ as { version?: number } | undefined;
  if (incomingMeta && typeof incomingMeta.version === "number" && incomingMeta.version !== current._metadata_.version) {
    throw err.conflict(
      `Version conflict: expected ${incomingMeta.version}, document is at version ${current._metadata_.version}`,
    );
  }

  const { _id_: _ignoredId, _metadata_: _ignoredMeta, collection: _ignoredCollection, ...changes } = patch;
  const merged: StoredDocument = {
    ...current,
    ...changes,
    collection: current.collection,
    _id_: current._id_,
  };

  const now = nowIso();
  merged._metadata_ = {
    checksum: await computeChecksum(withoutEnvelope(merged)),
    created: current._metadata_.created,
    updated: now,
    version: current._metadata_.version + 1,
  };
  return merged;
}

function withoutEnvelope(doc: StoredDocument): Record<string, unknown> {
  const { collection: _c, _id_: _i, _metadata_: _m, ...rest } = doc;
  return rest;
}

/**
 * Ensure a collection exists. The real server rejects writes to unknown
 * collections; the mock auto-registers a schema-less collection instead so
 * ad-hoc `api.collection("x")` usage works out of the box.
 */
export async function ensureCollection(tables: MockTables, name: string): Promise<void> {
  if (await tables.collections.get(name)) return;
  await tables.collections.put({
    name,
    version: "1.0.0",
    fields: {
      _id_: { name: "_id_", type: "string", required: true, unique: true },
      _metadata_: { name: "_metadata_", type: "object" },
    },
    _auto_created_: true,
  });
}

export async function deleteCollectionData(tables: MockTables, name: string): Promise<void> {
  const docs = await listDocs(tables, name);
  for (const doc of docs) {
    await tables.documents.delete([name, doc._id_]);
  }
  await tables.collections.delete(name);
}
