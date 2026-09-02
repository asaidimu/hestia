/**
 * Policy + IAM rule handlers.
 *
 * Policies are stored as `_operation_policy_` documents (so the client pager
 * can query them through the generic document query route); rules live in a
 * dedicated object store. Rule validation evaluates trivially — the mock has
 * no Cedar-style engine — but the wire contract is honored.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";
import { clone, nowIso, uuid } from "../../util";
import { requirePayload } from "../context";
import { POLICY_COLLECTION, applyUpdate, newDoc, putDoc, requireDoc } from "../store";

export function policyHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const { tables } = deps;

  const policyDoc = (raw: Record<string, unknown>) => ({
    operation: raw["operation"],
    key: raw["key"] ?? raw["operation"],
    rule: raw["rule"] ?? "",
    enabled: raw["enabled"] ?? true,
    protected: raw["protected"] ?? false,
    rateLimit: raw["rateLimit"],
    throttle: raw["throttle"],
  });

  return {
    "system:policies:policy:create": {
      access: "admin",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        const payload = requirePayload(ctx);
        if (await findByOperation(name)) throw err.duplicate(`policy "${name}"`);

        const doc = await newDoc(POLICY_COLLECTION, {
          _id_: name,
          operation: name,
          key: name,
          rule: String(payload["rule"] ?? "authenticated"),
          enabled: true,
          protected: false,
          rateLimit: payload["rateLimit"] ?? null,
          throttle: payload["throttle"] ?? null,
        });
        await putDoc(tables, doc);
        return ok(policyDoc(clone(doc)));
      },
    },

    "system:policies:policy:list": {
      access: "authenticated",
      handler: async () => {
        const docs = await tables.documents.getAllByIndex("by_collection", POLICY_COLLECTION);
        return ok({ policies: docs.map(policyDoc) });
      },
    },

    "system:policies:policy:update": {
      access: "admin",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        const current = await findByOperation(name);
        if (!current) throw err.notFound(`policy "${name}"`);

        const payload = requirePayload(ctx);
        const patch: Record<string, unknown> = {};
        for (const field of ["rule", "enabled", "rateLimit", "throttle"] as const) {
          if (payload[field] !== undefined) patch[field] = payload[field];
        }
        const updated = await applyUpdate(current, patch);
        await putDoc(tables, updated);
        return ok(policyDoc(clone(updated)));
      },
    },

    "system:policies:rule:create": {
      access: "admin",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        const payload = requirePayload(ctx);
        if (await tables.rules.get(name)) throw err.duplicate(`rule "${name}"`);
        const doc = {
          id: name,
          name,
          ruleType: String(payload["ruleType"] ?? "expression"),
          syntax: String(payload["syntax"] ?? "anansi-rule"),
          expression: payload["expression"] == null ? undefined : String(payload["expression"]),
          rules: payload["rules"],
          description: payload["description"] == null ? "" : String(payload["description"]),
          protected: false,
          created_at: nowIso(),
          updated_at: nowIso(),
        };
        await tables.rules.put(doc);
        return ok(clone(doc));
      },
    },

    "system:policies:rule:get": {
      access: "authenticated",
      handler: async (ctx) => {
        const doc = await tables.rules.get(ctx.args["name"]);
        if (!doc) throw err.notFound(`rule "${ctx.args["name"]}"`);
        return ok(clone(doc));
      },
    },

    "system:policies:rule:list": {
      access: "authenticated",
      handler: async () => {
        const docs = await tables.rules.getAll();
        return ok({ rules: docs });
      },
    },

    "system:policies:rule:update": {
      access: "admin",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        const current = await tables.rules.get(name);
        if (!current) throw err.notFound(`rule "${name}"`);
        const payload = requirePayload(ctx);
        const updated = {
          ...current,
          ...payload,
          name,
          updated_at: nowIso(),
        };
        await tables.rules.put(updated);
        return ok(clone(updated));
      },
    },

    "system:policies:rule:delete": {
      access: "admin",
      handler: async (ctx) => {
        const name = ctx.args["name"];
        const current = await tables.rules.get(name);
        if (!current) throw err.notFound(`rule "${name}"`);
        if (current["protected"]) throw err.denied("deleting a protected rule");
        await tables.rules.delete(name);
        return noContent();
      },
    },

    "system:policies:rule:validate": {
      access: "admin",
      handler: async (ctx) => {
        const payload = requirePayload(ctx);
        const rule = payload["rule"];
        if (rule == null) {
          return ok({ valid: false, error: "rule is required" });
        }
        // Trivial evaluator: accept literal booleans and "true"/"false"
        // expressions; everything else is considered well-formed but unknown.
        if (typeof rule === "boolean") return ok({ valid: true, result: rule });
        if (rule === "true") return ok({ valid: true, result: true });
        if (rule === "false") return ok({ valid: true, result: false });
        return ok({ valid: true, result: true, error: undefined });
      },
    },

    "system:policies:reload": {
      access: "admin",
      handler: async () => {
        const operations = await tables.documents.count();
        const rules = await tables.rules.count();
        return ok({ operations, rules });
      },
    },

    "system:policies:binding:get": {
      access: "admin",
      handler: async (ctx) => {
        const doc = await findByOperation(ctx.args["name"]);
        if (!doc) throw err.notFound(`binding "${ctx.args["name"]}"`);
        return ok({ operation: doc["operation"], rule: doc["rule"], enabled: doc["enabled"] });
      },
    },

    "system:policies:binding:list": {
      access: "admin",
      handler: async () => {
        const docs = await tables.documents.getAllByIndex("by_collection", POLICY_COLLECTION);
        return ok({
          data: docs.map((d) => ({ operation: d["operation"], rule: d["rule"], enabled: d["enabled"] })),
        });
      },
    },
  };

  async function findByOperation(operation: string) {
    return await tables.documents.get([POLICY_COLLECTION, operation]);
  }
}
