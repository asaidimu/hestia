/**
 * IndexedDB-backed `SimplePersistence` implementation for the Hestia client's
 * reactive auth store. Persisting `{identity: ...}` in the same database as
 * the mock server means the whole app state — session AND data — survives
 * reloads, enabling true offline-first development.
 */

import type { SimplePersistence } from "@asaidimu/utils-persistence";
import type { MockHestiaServer } from "./server/server";

const AUTH_STATE_KEY = "auth_state";

export interface IdbPersistenceConfig {
  /** App id embedded in the persistence metadata (default "hestia-mock"). */
  app?: string;
  /** Schema version reported to the store (default "1.0.0"). */
  version?: string;
}

/**
 * Create a `SimplePersistence<T>` backed by the mock server's `kv` store.
 *
 * @typeParam T the persisted state shape (typically `AuthState`).
 */
export function createIdbPersistence<T extends object>(
  server: MockHestiaServer,
  config: IdbPersistenceConfig = {},
): SimplePersistence<T> {
  const app = config.app ?? "hestia-mock";
  const version = config.version ?? "1.0.0";

  const listeners = new Map<string, Set<(state: T) => void>>();

  const read = async (): Promise<T | null> => {
    const record = (await server.tables.kv.get(AUTH_STATE_KEY)) as { v?: T } | undefined;
    return (record?.v ?? null) as T | null;
  };

  return {
    async set(_id: string, state: T): Promise<boolean> {
      try {
        await server.tables.kv.put({ k: AUTH_STATE_KEY, v: state });
        // Cross-instance notification (other tabs/components) — best effort.
        for (const [id, set] of listeners) {
          if (id === _id) continue;
          for (const listener of set) listener(state);
        }
        return true;
      } catch {
        return false;
      }
    },

    async get(): Promise<T | null> {
      try {
        return await read();
      } catch {
        return null;
      }
    },

    subscribe(id: string, callback: (state: T) => void): () => void {
      let set = listeners.get(id);
      if (!set) {
        set = new Set();
        listeners.set(id, set);
      }
      set.add(callback);
      return () => {
        set!.delete(callback);
        if (set!.size === 0) listeners.delete(id);
      };
    },

    async clear(): Promise<boolean> {
      try {
        await server.tables.kv.delete(AUTH_STATE_KEY);
        return true;
      } catch {
        return false;
      }
    },

    stats() {
      return { version, id: app };
    },
  } as SimplePersistence<T>;
}
