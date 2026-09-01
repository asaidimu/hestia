/**
 * Shared dependency bundle handed to every handler factory.
 */

import type { EventBus } from "../bus";
import type { MockTables } from "../schema";

export interface MockServerOptions {
  /** Simulated latency in ms, or a randomizer returning ms. */
  latency?: number | (() => number);
  /** Version string reported by the updates subsystem. */
  serverVersion?: string;
  /** When set, `updates:check` reports this as an available version. */
  updateAvailable?: string | null;
  /** Execute workflow runs with the stub engine (default true). */
  executeWorkflows?: boolean;
}

export interface ServerDeps {
  tables: MockTables;
  bus: EventBus;
  options: MockServerOptions;
  /** Wipe and re-seed the server state (system:core:reset). */
  reset: () => Promise<void>;
  /** Append an application log entry (used internally + by tests). */
  pushLog: (entry: { level: string; msg: string; caller?: string; fields?: Record<string, unknown> }) => Promise<void>;
  /** Mark bootstrap complete (admin credentials exist). */
  setBootstrapped: (value: boolean) => Promise<void>;
  isBootstrapped: () => Promise<boolean>;
}
