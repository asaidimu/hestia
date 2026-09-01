/**
 * Evaluator for the `@asaidimu/query` DSL — the same query language the real
 * Hestia server executes (see core/system/collections/query.go).
 *
 * Supported (the "standard" tier):
 *  - every comparison operator: eq, neq, lt, lte, gt, gte, in, nin,
 *    contains, ncontains, startswith, endswith, exists, nexists
 *  - logical groups: and, or, not, nor, xor (nested arbitrarily)
 *  - multi-field sort with per-field direction
 *  - offset pagination (cursor pagination degrades to a full scan + offset 0)
 *  - dotted field paths (`a.b.c`) and array indices (`items.0`)
 */

import type {
  QueryDSL,
  QueryFilter,
  FilterCondition,
  FilterGroup,
  SortConfiguration,
} from "@asaidimu/query";
import { compareValues, deepEqual } from "./util";

/** Resolve a possibly-dotted path against a document. */
export function getField(doc: unknown, path: string): unknown {
  if (!path.includes(".")) return (doc as Record<string, unknown>)?.[path];
  const parts = path.split(".");
  let cur: unknown = doc;
  for (const part of parts) {
    if (cur == null) return undefined;
    if (Array.isArray(cur)) {
      const idx = Number(part);
      cur = Number.isInteger(idx) ? cur[idx] : undefined;
      continue;
    }
    cur = (cur as Record<string, unknown>)[part];
  }
  return cur;
}

function asString(v: unknown): string | null {
  if (v == null) return null;
  if (typeof v === "string") return v;
  if (typeof v === "number" || typeof v === "boolean") return String(v);
  return null;
}

function evaluateCondition(doc: Record<string, unknown>, cond: FilterCondition): boolean {
  const actual = getField(doc, cond.field);
  const expected = cond.value;

  switch (cond.operator) {
    case "eq":
      return deepEqual(actual, expected);
    case "neq":
      return !deepEqual(actual, expected);
    case "lt":
      return actual != null && expected != null && compareValues(actual, expected) < 0;
    case "lte":
      return actual != null && expected != null && compareValues(actual, expected) <= 0;
    case "gt":
      return actual != null && expected != null && compareValues(actual, expected) > 0;
    case "gte":
      return actual != null && expected != null && compareValues(actual, expected) >= 0;
    case "in": {
      const list = Array.isArray(expected) ? expected : [expected];
      return list.some((v) => deepEqual(actual, v));
    }
    case "nin": {
      const list = Array.isArray(expected) ? expected : [expected];
      return !list.some((v) => deepEqual(actual, v));
    }
    case "contains": {
      if (Array.isArray(actual)) return actual.some((v) => deepEqual(v, expected));
      const s = asString(actual);
      const needle = asString(expected);
      return s != null && needle != null && s.includes(needle);
    }
    case "ncontains": {
      if (Array.isArray(actual)) return !actual.some((v) => deepEqual(v, expected));
      const s = asString(actual);
      const needle = asString(expected);
      return s == null || needle == null || !s.includes(needle);
    }
    case "startswith": {
      const s = asString(actual);
      const prefix = asString(expected);
      return s != null && prefix != null && s.startsWith(prefix);
    }
    case "endswith": {
      const s = asString(actual);
      const suffix = asString(expected);
      return s != null && suffix != null && s.endsWith(suffix);
    }
    case "exists":
      return expected === false ? actual === undefined : actual !== undefined;
    case "nexists":
      return actual === undefined;
    default:
      // Unknown operator — treat as non-match rather than throwing, so a
      // typo in a filter degrades gracefully in a mock.
      return false;
  }
}

function isGroup(filter: QueryFilter): filter is FilterGroup {
  return typeof (filter as FilterGroup).operator === "string" && Array.isArray((filter as FilterGroup).conditions);
}

export function evaluateFilter(doc: Record<string, unknown>, filter: QueryFilter): boolean {
  if (isGroup(filter)) {
    const results = filter.conditions.map((c) => evaluateFilter(doc, c as QueryFilter));
    switch (filter.operator) {
      case "and":
        return results.every(Boolean);
      case "or":
        return results.some(Boolean);
      case "not":
        return !results.every(Boolean);
      case "nor":
        return !results.some(Boolean);
      case "xor":
        return results.filter(Boolean).length % 2 === 1;
      default:
        return results.every(Boolean);
    }
  }
  return evaluateCondition(doc, filter as FilterCondition);
}

/** Multi-field sort. Missing values sort last regardless of direction. */
export function sortDocs<T extends Record<string, unknown>>(
  docs: T[],
  sort: SortConfiguration[] | undefined,
): T[] {
  if (!sort || sort.length === 0) return docs;
  const specs = sort.map((s) => ({
    field: String(s.field),
    dir: (s.direction === "desc" ? -1 : 1),
  }));

  return [...docs].sort((a, b) => {
    for (const spec of specs) {
      const av = getField(a, spec.field);
      const bv = getField(b, spec.field);
      if (av == null && bv == null) continue;
      if (av == null) return 1;
      if (bv == null) return -1;
      const cmp = compareValues(av, bv);
      if (cmp !== 0) return cmp * spec.dir;
    }
    return 0;
  });
}

export interface QueryOutcome<T> {
  items: T[];
  total: number;
}

/** Execute a full DSL query against an in-memory array of documents. */
export function executeQuery<T extends Record<string, unknown>>(
  docs: T[],
  dsl: QueryDSL<T> | Record<string, unknown> | undefined | null,
): QueryOutcome<T> {
  const query = (dsl ?? {}) as QueryDSL<T>;
  let out = docs;

  if (query.filters) {
    out = out.filter((d) => evaluateFilter(d, query.filters as QueryFilter));
  }

  const total = out.length;

  if (query.sort?.length) {
    out = sortDocs(out, query.sort as SortConfiguration[]);
  }

  const pagination = query.pagination as
    | { type: "offset"; offset?: number; limit?: number }
    | { type: "cursor"; cursor?: string; limit?: number; direction?: string }
    | undefined;

  let offset = 0;
  let limit = out.length;
  if (pagination) {
    if (pagination.type === "offset") {
      offset = Math.max(0, pagination.offset ?? 0);
      limit = pagination.limit ?? out.length;
    } else {
      // Cursor pagination: degrade gracefully (mock keeps natural order).
      offset = 0;
      limit = pagination.limit ?? out.length;
    }
  }

  const items = out.slice(offset, offset + limit);
  return { items, total };
}

/** Build the page metadata the client expects under `metadata.page`. */
export function paginationFor(total: number, offset: number, limit: number, count: number): {
  number: number;
  size: number;
  count: number;
  total: number;
  pages: number;
} {
  const size = Math.max(1, limit);
  return {
    number: Math.floor(Math.max(0, offset) / size) + 1,
    size,
    count,
    total,
    pages: Math.max(1, Math.ceil(total / size)),
  };
}
