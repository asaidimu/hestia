/**
 * Self-update subsystem + application log handlers.
 */

import type { RouteSpec } from "../context";
import type { ServerDeps } from "../deps";
import { err } from "../../errors";
import { ok, noContent } from "../../envelope";

interface UpdateState {
  staged_version: string;
  prepared: boolean;
  last_check: number;
}

async function getUpdateState(deps: ServerDeps): Promise<UpdateState> {
  const raw = (await deps.tables.kv.get("update_state")) as { v?: UpdateState } | undefined;
  return raw?.v ?? { staged_version: "", prepared: false, last_check: 0 };
}

async function putUpdateState(deps: ServerDeps, state: UpdateState): Promise<void> {
  await deps.tables.kv.put({ k: "update_state", v: state });
}

export function updateHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  const version = () => deps.options.serverVersion ?? "1.0.1-mock";

  return {
    "system:updates:status:get": {
      access: "admin",
      handler: async () => {
        const state = await getUpdateState(deps);
        return ok({
          version: version(),
          staged_version: state.staged_version,
          prepared: state.prepared,
          last_check: state.last_check,
        });
      },
    },

    "system:updates:changelog:get": {
      access: "admin",
      handler: async () => {
        const state = await getUpdateState(deps);
        if (!state.staged_version) {
          throw err.notFound("no staged update changelog");
        }
        return ok({
          version: state.staged_version,
          asset_name: `hestia-${state.staged_version}.tar.gz`,
          changelog: `# ${state.staged_version}\n\nMock release notes.`,
        });
      },
    },

    "system:updates:check:get": {
      access: "admin",
      handler: async () => {
        const available = deps.options.updateAvailable ?? null;
        return ok({ available: !!available, version: available ?? version() });
      },
    },

    "system:updates:check:create": {
      access: "admin",
      handler: async () => {
        const available = deps.options.updateAvailable ?? null;
        const state = await getUpdateState(deps);
        state.last_check = Date.now();
        let staged = false;
        if (available) {
          state.staged_version = available;
          state.prepared = true;
          staged = true;
        }
        await putUpdateState(deps, state);
        return ok({ checked: true, staged, version: available ?? version(), auto_apply: false });
      },
    },

    "system:updates:stage:create": {
      access: "admin",
      handler: async () => {
        const available = deps.options.updateAvailable ?? null;
        const state = await getUpdateState(deps);
        if (!available) {
          await putUpdateState(deps, { ...state, last_check: Date.now() });
          return ok({ staged: false, version: version() });
        }
        state.staged_version = available;
        state.prepared = true;
        state.last_check = Date.now();
        await putUpdateState(deps, state);
        return ok({ staged: true, version: available });
      },
    },

    "system:updates:update:apply": {
      access: "admin",
      handler: async () => {
        const state = await getUpdateState(deps);
        if (!state.staged_version || !state.prepared) {
          throw err.validation("no staged update to apply");
        }
        return ok({ message: `update to ${state.staged_version} prepared; restart required` });
      },
    },

    "system:updates:update:discard": {
      access: "admin",
      handler: async () => {
        await putUpdateState(deps, { staged_version: "", prepared: false, last_check: Date.now() });
        return noContent();
      },
    },
  };
}

interface LogQueryPayload {
  level?: string;
  from?: string;
  to?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export function logHandlers(deps: ServerDeps): Record<string, RouteSpec> {
  return {
    "system:logs:list": {
      access: "admin",
      noAudit: true,
      handler: async (ctx) => {
        const q = (ctx.payload ?? {}) as LogQueryPayload;
        let entries = await deps.tables.logs.getAll();

        if (q.level) entries = entries.filter((e) => e.level === q.level);
        if (q.from) entries = entries.filter((e) => e.ts >= Date.parse(q.from!));
        if (q.to) entries = entries.filter((e) => e.ts <= Date.parse(q.to!));
        if (q.search) {
          const needle = q.search.toLowerCase();
          entries = entries.filter((e) => (e.msg ?? "").toLowerCase().includes(needle));
        }

        entries.sort((a, b) => b.ts - a.ts);

        const offset = Math.max(0, q.offset ?? 0);
        const limit = Math.max(1, q.limit ?? 100);
        const slice = entries.slice(offset, offset + limit);

        return ok({
          entries: slice,
          total: entries.length,
          has_more: offset + limit < entries.length,
        });
      },
    },

    "system:scheduler:job:list": {
      access: "authenticated",
      handler: async () => {
        const schedules = await deps.tables.documents.getAllByIndex("by_collection", "_scheduled_messages_");
        return ok({
          data: schedules
            .filter((s) => !s["disabled"])
            .map((s) => ({
              id: s["_id_"],
              message: s["message"],
              cron: s["cron"],
              next_run: null,
            })),
        });
      },
    },
  };
}
