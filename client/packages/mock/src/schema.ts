/**
 * Database schema for the mock Hestia server.
 *
 * All server state lives in IndexedDB: documents, sessions, blobs, workflows,
 * settings, logs — everything. Closing the tab and reopening later restores a
 * fully functional "server".
 */

import { openDatabase, table, withTransaction, Store, type Table } from "./idb";

export const DEFAULT_DB_NAME = "hestia-mock";
export const DB_VERSION = 1;

/** Every object store in the mock server, with its key configuration. */
export const STORES = {
  /** Generic key-value store (seed flags, update state, pointers). */
  kv: { keyPath: "k" },
  /** Active sessions: {token, user_id, created_at, expires_at}. */
  sessions: { keyPath: "token", indexes: [{ name: "by_user", keyPath: "user_id" }] },
  /** Collection metadata (anansi SchemaDefinition documents). */
  collections: { keyPath: "name" },
  /** Every document in every collection — system and user — keyed [collection, _id_]. */
  documents: {
    keyPath: ["collection", "_id_"],
    indexes: [{ name: "by_collection", keyPath: "collection" }],
  },
  /** Blob namespaces. */
  namespaces: { keyPath: "ns" },
  /** Stored blobs: {ns, key, meta, data}. */
  blobs: { keyPath: ["ns", "key"], indexes: [{ name: "by_ns", keyPath: "ns" }] },
  /** Resumable/staged upload sessions. */
  upload_sessions: { keyPath: "session_id" },
  /** IAM policy rules (raw PolicyRule records). */
  rules: { keyPath: "id" },
  /** In-app notifications. */
  notifications: { keyPath: "_id_", indexes: [{ name: "by_user", keyPath: "user_id" }] },
  /** Application log entries. */
  logs: { keyPath: "id", indexes: [{ name: "by_level", keyPath: "level" }] },
  /** Cron schedules. */
  schedules: { keyPath: "_id_", indexes: [{ name: "by_user", keyPath: "user_id" }] },
  /** Workflow definitions. */
  workflows: { keyPath: "_id_" },
  /** Workflow runs. */
  runs: { keyPath: "run_id" },
  /** Workflow timeline events keyed [run_id, seq]. */
  run_events: { keyPath: ["run_id", "seq"], indexes: [{ name: "by_run", keyPath: "run_id" }] },
  /** Platform capabilities. */
  capabilities: { keyPath: "name" },
  /** Password reset tokens. */
  password_resets: { keyPath: "token", indexes: [{ name: "by_email", keyPath: "email" }] },
} as const;

export type StoreName = keyof typeof STORES;

/** Convenience accessor grouping typed tables. */
export interface MockTables {
  kv: Table<{ k: string; v: unknown }, string>;
  sessions: Table<SessionRecord, string>;
  collections: Table<Record<string, unknown>, string>;
  documents: Table<StoredDocument, [string, string]>;
  namespaces: Table<NamespaceRecord, string>;
  blobs: Table<BlobRecord, [string, string]>;
  upload_sessions: Table<UploadSessionRecord, string>;
  rules: Table<RuleRecord, string>;
  notifications: Table<Record<string, unknown>, string>;
  logs: Table<LogRecord, string>;
  schedules: Table<Record<string, unknown>, string>;
  workflows: Table<Record<string, unknown>, string>;
  runs: Table<RunRecord, string>;
  run_events: Table<RunEventRecord, [string, number]>;
  capabilities: Table<Record<string, unknown>, string>;
  password_resets: Table<ResetTokenRecord, string>;
}

export interface SessionRecord {
  token: string;
  user_id: string;
  created_at: number;
  expires_at: number;
}

export interface StoredDocument {
  collection: string;
  _id_: string;
  _metadata_: {
    checksum: string;
    created: string;
    updated: string;
    version: number;
  };
  [field: string]: unknown;
}

export interface NamespaceRecord {
  ns: string;
  id: string;
  display_name: string;
  custom?: Record<string, string>;
  created_at: string;
}

export interface BlobRecord {
  ns: string;
  key: string;
  meta: {
    key: string;
    namespace_id: string;
    content_type: string;
    size: number;
    created_at: string;
    updated_at?: string;
    custom?: Record<string, any>;
  };
  data: Uint8Array;
}

export interface UploadSessionRecord {
  session_id: string;
  ns: string;
  key: string;
  expected_size: number;
  content_type: string;
  block_size: number;
  overwrite: boolean;
  chunks: { offset: number; data: Uint8Array }[];
  received: number;
  created_at: number;
}

export interface RuleRecord {
  id: string;
  name: string;
  ruleType?: string;
  syntax?: string;
  expression?: string;
  rules?: unknown;
  description?: string;
  protected?: boolean;
  created_at: string;
  updated_at: string;
}

export interface LogRecord {
  id: string;
  level: string;
  ts: number;
  caller: string;
  msg: string;
  fields?: Record<string, unknown>;
  [extra: string]: unknown;
}

export interface RunRecord {
  run_id: string;
  pipeline_id: string;
  start_time: number;
  end_time?: number;
  event_count: number;
  status: "recording" | "complete" | "failed" | "paused";
  metadata?: Record<string, unknown>;
  final_state?: Record<string, unknown>;
  error?: string;
}

export interface RunEventRecord {
  run_id: string;
  seq: number;
  timestamp: number;
  source: string;
  type: string;
  payload: Record<string, unknown>;
  delta?: Record<string, unknown>;
}

export interface ResetTokenRecord {
  token: string;
  email: string;
  created_at: number;
  expires_at: number;
}

function createStores(db: IDBDatabase): void {
  for (const [name, def] of Object.entries(STORES)) {
    if (db.objectStoreNames.contains(name)) continue;
    const store = db.createObjectStore(name, { keyPath: def.keyPath as string | string[] });
    for (const index of (def as { indexes?: { name: string; keyPath: string | string[] }[] }).indexes ?? []) {
      store.createIndex(index.name, index.keyPath as string | string[]);
    }
  }
}

/** Open (and lazily create) the mock server database. */
export async function openMockDatabase(name: string = DEFAULT_DB_NAME): Promise<{ db: IDBDatabase; tables: MockTables }> {
  const db = await openDatabase({
    name,
    version: DB_VERSION,
    upgrade: (_db, _old) => createStores(_db),
  });

  return {
    db,
    tables: {
      kv: table(db, "kv"),
      sessions: table(db, "sessions"),
      collections: table(db, "collections"),
      documents: table(db, "documents"),
      namespaces: table(db, "namespaces"),
      blobs: table(db, "blobs"),
      upload_sessions: table(db, "upload_sessions"),
      rules: table(db, "rules"),
      notifications: table(db, "notifications"),
      logs: table(db, "logs"),
      schedules: table(db, "schedules"),
      workflows: table(db, "workflows"),
      runs: table(db, "runs"),
      run_events: table(db, "run_events"),
      capabilities: table(db, "capabilities"),
      password_resets: table(db, "password_resets"),
    },
  };
}

export const ALL_STORE_NAMES: string[] = Object.keys(STORES);

/** Wipe every object store (used by `system:core:reset` and tests). */
export async function wipeAllStores(db: IDBDatabase): Promise<void> {
  const names = ALL_STORE_NAMES.filter((n) => db.objectStoreNames.contains(n));
  await withTransaction(db, names, "readwrite", async (tx) => {
    await Promise.all(names.map((n) => new Store(tx, n).clear()));
  });
}
