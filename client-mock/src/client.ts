/**
 * One-call wiring: a fully-functional `HestiaClient` whose backend is
 * IndexedDB. Auth identity is persisted through the same database, so login
 * state and all data survive reloads.
 *
 * ```ts
 * const mock = await createMockHestia({ seed: { email: "admin@x.dev", password: "s3cret" } });
 * await mock.client.auth.login("admin@x.dev", "s3cret");
 * await mock.client.collection("todos").create({ data: { title: "ship it" } });
 * ```
 */

import { HestiaClient } from "@asaidimu/hestia";
import type { UserIdentity } from "@asaidimu/hestia";
import { MockHestiaServer } from "./server/server";
import type { MockHestiaServerConfig } from "./server/server";
import type { MockServerOptions } from "./server/deps";
import { IndexedDbTransport } from "./transport";
import { createIdbPersistence } from "./persistence";

export interface MockHestiaOptions extends MockHestiaServerConfig {
  /** Options forwarded to the mock server (latency, updates, workflows...). */
  serverOptions?: MockServerOptions;
}

export interface MockHestia {
  /** The real SDK client — `api.auth`, `api.users`, `api.collection(...)` etc. */
  client: HestiaClient;
  /** The transport implementing the SDK `Transport` interface. */
  transport: IndexedDbTransport;
  /** The IndexedDB-backed mock server. */
  server: MockHestiaServer;
  /** Convenience passthrough to `server.bus` for injecting stream events. */
  destroy: () => Promise<void>;
}

/**
 * Create a mock-backed Hestia client.
 *
 * Every call the client makes is served from IndexedDB; sessions, documents,
 * blobs, and streams all behave like the real server. When the mock answers
 * 401 the client's identity is cleared automatically (mirroring HttpTransport).
 */
export async function createMockHestia(options: MockHestiaOptions = {}): Promise<MockHestia> {
  const server = await MockHestiaServer.create({
    database: options.database,
    seed: options.seed,
    reset: options.reset,
    options: options.serverOptions ?? options.options,
  });

  const transport = new IndexedDbTransport(server);
  const persistence = createIdbPersistence<{ identity: UserIdentity | null }>(server);

  const client = new HestiaClient({
    // HestiaConfig requires a baseUrl; the transport ignores it.
    baseUrl: transport.base(),
    transport: transport as unknown as HestiaClient["client"],
    persistence,
  });

  transport.setOnUnauthorized(() => {
    void client.store.set({ identity: null });
  });

  return {
    client,
    transport,
    server,
    destroy: async () => {
      await server.close();
    },
  };
}
