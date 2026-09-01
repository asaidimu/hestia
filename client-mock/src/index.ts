/**
 * @asaidimu/hestia-mock — IndexedDB-backed mock server + transport for the
 * Hestia TypeScript client SDK.
 *
 * @example
 * ```ts
 * import { createMockHestia } from "@asaidimu/hestia-mock";
 *
 * const mock = await createMockHestia();
 * await mock.client.auth.login("admin@test.local", "password123");
 *
 * const todos = mock.client.collection<{ title: string }>("todos");
 * await todos.create({ data: { title: "try the mock" } });
 * ```
 */

// Client wiring
export { createMockHestia } from "./client";
export type { MockHestia, MockHestiaOptions } from "./client";

// Transport
export { IndexedDbTransport } from "./transport";
export type { IndexedDbTransportOptions } from "./transport";

// Mock server (direct dispatch, seeding, manual schedule firing, logs)
export { MockHestiaServer } from "./server/server";
export type { MockHestiaServerConfig } from "./server/server";
export type { MockServerOptions } from "./server/deps";

// Persistence (IDB-backed SimplePersistence for the client auth store)
export { createIdbPersistence } from "./persistence";
export type { IdbPersistenceConfig } from "./persistence";

// Query engine (exposed for testing/inspection)
export { executeQuery, evaluateFilter, sortDocs, getField } from "./query";

// Event bus (realtime stream source)
export { EventBus, Topics } from "./bus";

// Errors (SystemError-compatible factories with HTTP status mapping)
export { MockHttpError, err as mockErrors, statusForError } from "./errors";

// Database schema helpers
export {
  DEFAULT_DB_NAME,
  openMockDatabase,
  wipeAllStores,
  STORES,
  type MockTables,
  type StoredDocument,
} from "./schema";

// Seeding
export { seedDatabase, DEFAULT_SEED, type SeedConfig } from "./seed";
