/**
 * First-run seeding: administrator account, IAM rules, operation policies,
 * capabilities, settings, and core docs — mirroring what the real server
 * provisions on first boot.
 */

import type { MockTables } from "./schema";
import { hashPassword, nowIso, uuid } from "./util";
import { newDoc, putDoc, USER_COLLECTION, POLICY_COLLECTION } from "./server/store";

export interface SeedConfig {
  email: string;
  password: string;
  name?: string;
  /** Grant the seeded user the `administrator` permission (default true). */
  admin?: boolean;
  permissions?: string[];
  tenantId?: string;
}

export const DEFAULT_SEED: SeedConfig = {
  email: "admin@test.local",
  password: "password123",
  name: "Administrator",
  admin: true,
  permissions: ["administrator", "authenticated"],
  tenantId: "root",
};

const IAM_RULES = [
  { id: "administrator", name: "administrator", description: "Full administrative access", expression: "true" },
  { id: "authenticated", name: "authenticated", description: "Any authenticated session", expression: "true" },
  { id: "owner", name: "owner", description: "Owner of the resource", expression: "identity.user._id_ == resource.user_id" },
  { id: "anonymous", name: "anonymous", description: "Unauthenticated access", expression: "false" },
];

const OPERATION_POLICIES: { name: string; rule: string }[] = [
  { name: "system:auth:session:create", rule: "anonymous" },
  { name: "system:auth:session:delete", rule: "authenticated" },
  { name: "system:auth:password:reset", rule: "anonymous" },
  { name: "system:auth:password:confirm", rule: "anonymous" },
  { name: "system:users:user:create", rule: "anonymous" },
  { name: "system:users:user:get", rule: "authenticated" },
  { name: "system:users:user:update", rule: "authenticated" },
  { name: "system:users:user:delete", rule: "administrator" },
  { name: "system:users:password:change", rule: "authenticated" },
  { name: "system:collections:collection:list", rule: "authenticated" },
  { name: "system:collections:document:create", rule: "authenticated" },
  { name: "system:collections:document:query", rule: "authenticated" },
  { name: "system:apikeys:key:create", rule: "authenticated" },
  { name: "system:apikeys:key:list", rule: "authenticated" },
  { name: "system:apikeys:key:rotate", rule: "authenticated" },
  { name: "system:notifications:notification:create", rule: "administrator" },
  { name: "system:notifications:notification:list", rule: "authenticated" },
  { name: "system:schedules:schedule:create", rule: "authenticated" },
  { name: "system:schedules:schedule:all", rule: "administrator" },
  { name: "system:audit:log:export", rule: "administrator" },
  { name: "system:audit:log:stream", rule: "administrator" },
  { name: "system:logs:list", rule: "administrator" },
  { name: "system:updates:status:get", rule: "administrator" },
  { name: "system:core:health:check", rule: "anonymous" },
  { name: "system:core:heartbeat", rule: "authenticated" },
  { name: "system:core:reset", rule: "administrator" },
];

const CAPABILITIES = [
  { name: "collections", display_name: "Collections", enabled: true, version: "1.0.0" },
  { name: "blobs", display_name: "Blob Storage", enabled: true, version: "1.0.0" },
  { name: "workflows", display_name: "Workflows (hermes)", enabled: true, version: "1.0.0" },
  { name: "schedules", display_name: "Schedules", enabled: true, version: "1.0.0" },
  { name: "notifications", display_name: "Notifications", enabled: true, version: "1.0.0" },
  { name: "updates", display_name: "Self-Update", enabled: false, version: "1.0.0" },
];

const SETTINGS = [
  { key: "platform", value: { name: "hestia-mock", version: "1.0.1", environment: "mock" } },
  { key: "mailer", value: { enabled: false, from: "mock@hestia.local" } },
  { key: "security", value: { session_ttl_days: 7, password_min_length: 8 } },
];

const CORE_DOCS = [
  { _id_: "doc-getting-started", title: "Getting Started", body: "Create a collection, then read and write documents." },
  { _id_: "doc-collections", title: "Collections", body: "Schema-less document storage with a rich query DSL." },
  { _id_: "doc-blobs", title: "Blobs", body: "Namespaced binary storage with resumable uploads." },
];

/** System collections that must be queryable through the generic document routes. */
const SYSTEM_COLLECTIONS = [
  "_user_",
  "_api_key_",
  "_operation_policy_",
  "_audit_log_",
  "_scheduled_messages_",
  "_workflow_definitions_",
  "_settings_",
];

/** Seed a fresh database. Idempotent: skips when the admin user exists. */
export async function seedDatabase(tables: MockTables, config: SeedConfig): Promise<void> {
  const existingUsers = await tables.documents.getAllByIndex("by_collection", USER_COLLECTION);
  if (existingUsers.length > 0) return;

  // Register system collection metadata so document queries work on them.
  for (const name of SYSTEM_COLLECTIONS) {
    if (!(await tables.collections.get(name))) {
      await tables.collections.put({
        name,
        version: "1.0.0",
        fields: {
          _id_: { name: "_id_", type: "string", required: true, unique: true },
          _metadata_: { name: "_metadata_", type: "object" },
        },
        _system_: true,
      });
    }
  }

  const admin = await newDoc(USER_COLLECTION, {
    email: config.email,
    name: config.name ?? "Administrator",
    password: await hashPassword(config.password),
    verified: true,
    permissions: config.permissions ?? (config.admin !== false ? ["administrator"] : ["authenticated"]),
    tenant_id: config.tenantId ?? "root",
    deleted: null,
  });
  await putDoc(tables, admin);

  for (const rule of IAM_RULES) {
    await tables.rules.put({
      ...rule,
      ruleType: "expression",
      syntax: "anansi-rule",
      protected: rule.id === "administrator" || rule.id === "authenticated",
      created_at: nowIso(),
      updated_at: nowIso(),
    });
  }

  for (const policy of OPERATION_POLICIES) {
    await tables.documents.put(await newDoc(POLICY_COLLECTION, {
      _id_: policy.name,
      operation: policy.name,
      key: policy.name,
      rule: policy.rule,
      enabled: true,
      protected: true,
      rateLimit: null,
      throttle: null,
    }));
  }

  for (const capability of CAPABILITIES) {
    await tables.capabilities.put({ ...capability, id: uuid() });
  }

  for (const setting of SETTINGS) {
    await tables.documents.put(await newDoc("_settings_", { _id_: setting.key, ...setting }));
  }

  await tables.kv.put({ k: "core_docs", v: CORE_DOCS });
}
