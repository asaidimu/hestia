/**
 * Minimal promise-based IndexedDB wrapper.
 *
 * Zero dependencies, works in browsers, Node (via `fake-indexeddb`), and any
 * environment exposing the standard `indexedDB` global. Deliberately thin —
 * every operation opens a short-lived transaction so concurrent callers never
 * fight over aborted transactions.
 */

export type Key = string | number | Date | BufferSource | IDBValidKey[];
export type Mode = IDBTransactionMode;

/** Thrown when IndexedDB is unavailable or a request fails at the event level. */
export class IdbError extends Error {
  constructor(message: string, public override readonly cause?: unknown) {
    super(message);
    this.name = "IdbError";
  }
}

export interface OpenOptions {
  name: string;
  version: number;
  /** Called inside `onupgradeneeded` to create/upgrade object stores. */
  upgrade: (db: IDBDatabase, oldVersion: number, tx: IDBTransaction) => void;
}

function reqToPromise<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(new IdbError("IndexedDB request failed", req.error));
  });
}

/** Open (and upgrade) a database, caching the connection. */
export async function openDatabase(opts: OpenOptions): Promise<IDBDatabase> {
  if (typeof indexedDB === "undefined") {
    throw new IdbError(
      "IndexedDB is not available in this environment. In Node tests add `fake-indexeddb`.",
    );
  }
  return await new Promise<IDBDatabase>((resolve, reject) => {
    const req = indexedDB.open(opts.name, opts.version);
    req.onupgradeneeded = (event) => {
      const db = req.result;
      const tx = req.transaction!;
      opts.upgrade(db, event.oldVersion, tx);
    };
    req.onsuccess = () => {
      const db = req.result;
      db.onversionchange = () => db.close();
      resolve(db);
    };
    req.onerror = () => reject(new IdbError(`Failed to open database "${opts.name}"`, req.error));
    req.onblocked = () => reject(new IdbError(`Database "${opts.name}" is blocked by another connection`));
  });
}

/**
 * Run `fn` inside a transaction on the given stores. The transaction commits
 * when the returned promise settles (via `txn.oncomplete`) — callers only need
 * to perform requests inside `fn`.
 */
export async function withTransaction<R>(
  db: IDBDatabase,
  stores: string[],
  mode: Mode,
  fn: (tx: IDBTransaction) => Promise<R>,
): Promise<R> {
  const tx = db.transaction(stores, mode);
  const done = new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onabort = () => reject(new IdbError("Transaction aborted", tx.error));
    tx.onerror = () => reject(new IdbError("Transaction failed", tx.error));
  });

  let result: R;
  try {
    result = await fn(tx);
  } catch (err) {
    // Swallow the abort rejection so it doesn't surface as unhandled.
    done.catch(() => undefined);
    try { tx.abort(); } catch { /* already aborted */ }
    throw err;
  }
  await done;
  return result;
}

/** Typed helpers bound to one object store within an existing transaction. */
export class Store {
  constructor(
    private readonly tx: IDBTransaction,
    public readonly name: string,
  ) {}

  private get raw(): IDBObjectStore {
    return this.tx.objectStore(this.name);
  }

  get(key: IDBValidKey): Promise<any> {
    return reqToPromise(this.raw.get(key));
  }

  getAll(query?: IDBValidKey | IDBKeyRange, count?: number): Promise<any[]> {
    return reqToPromise(this.raw.getAll(query, count));
  }

  getAllByIndex(index: string, query?: IDBValidKey | IDBKeyRange, count?: number): Promise<any[]> {
    return reqToPromise(this.raw.index(index).getAll(query, count));
  }

  count(query?: IDBValidKey | IDBKeyRange): Promise<number> {
    return reqToPromise(this.raw.count(query));
  }

  getKey(query: IDBValidKey | IDBKeyRange): Promise<IDBValidKey | undefined> {
    return reqToPromise(this.raw.getKey(query));
  }

  put(value: any, key?: IDBValidKey): Promise<IDBValidKey> {
    return reqToPromise(this.raw.put(value, key));
  }

  add(value: any, key?: IDBValidKey): Promise<IDBValidKey> {
    return reqToPromise(this.raw.add(value, key));
  }

  delete(key: IDBValidKey): Promise<undefined> {
    return reqToPromise(this.raw.delete(key));
  }

  clear(): Promise<undefined> {
    return reqToPromise(this.raw.clear());
  }

  /** Iterate all values with a cursor (supports range + direction). */
  async forEach(
    opts: { query?: IDBValidKey | IDBKeyRange; direction?: IDBCursorDirection; index?: string },
    visit: (value: any, cursorKey: IDBValidKey) => void | Promise<void>,
  ): Promise<void> {
    const source = opts.index ? this.raw.index(opts.index) : this.raw;
    const cursorReq = source.openCursor(opts.query, opts.direction);
    return await new Promise<void>((resolve, reject) => {
      cursorReq.onsuccess = async () => {
        const cursor = cursorReq.result;
        if (!cursor) return resolve();
        try {
          await visit(cursor.value, cursor.key);
          cursor.continue();
        } catch (err) {
          reject(err);
        }
      };
      cursorReq.onerror = () => reject(new IdbError("Cursor failed", cursorReq.error));
    });
  }
}

/** A thin typed table facade over one object store. */
export class Table<T = any, K extends Key = Key> {
  constructor(
    private readonly db: IDBDatabase,
    public readonly name: string,
  ) {}

  async get(key: K): Promise<T | undefined> {
    return await withTransaction(this.db, [this.name], "readonly", async (tx) =>
      await new Store(tx, this.name).get(key),
    );
  }

  async getAll(): Promise<T[]> {
    return await withTransaction(this.db, [this.name], "readonly", async (tx) =>
      await new Store(tx, this.name).getAll(),
    );
  }

  async getAllByIndex<R = T>(index: string, query?: IDBValidKey | IDBKeyRange): Promise<R[]> {
    return await withTransaction(this.db, [this.name], "readonly", async (tx) =>
      await new Store(tx, this.name).getAllByIndex(index, query),
    );
  }

  async count(query?: IDBValidKey | IDBKeyRange): Promise<number> {
    return await withTransaction(this.db, [this.name], "readonly", async (tx) =>
      await new Store(tx, this.name).count(query),
    );
  }

  async put(value: T): Promise<void> {
    await withTransaction(this.db, [this.name], "readwrite", async (tx) => {
      await new Store(tx, this.name).put(value);
    });
  }

  async delete(key: K): Promise<void> {
    await withTransaction(this.db, [this.name], "readwrite", async (tx) => {
      await new Store(tx, this.name).delete(key as unknown as IDBValidKey);
    });
  }

  async clear(): Promise<void> {
    await withTransaction(this.db, [this.name], "readwrite", async (tx) => {
      await new Store(tx, this.name).clear();
    });
  }

  /** Low-level access for multi-store transactions. */
  store(tx: IDBTransaction): Store {
    return new Store(tx, this.name);
  }
}

export function table<T = any>(db: IDBDatabase, name: string): Table<T> {
  return new Table<T>(db, name);
}
