import type { Document, DocumentMetadata, Page } from "./types";

/** Methods attached to every document via {@link proxiedDocument}. */
export interface DocumentMethods {
  /** The document's envelope id (`_id_`). */
  id(): string;
  /** The document's envelope metadata (`_metadata_`). */
  metadata(): DocumentMetadata;
}

/** A document with {@link DocumentMethods} attached. */
export type ProxiedDocument<T extends Record<string, any>> = Document<T> & DocumentMethods;

const METHOD_NAMES = new Set<PropertyKey>(["id", "metadata"]);

/**
 * Wraps a document in a Proxy exposing `id()` and `metadata()` alongside
 * every payload field. Property reads/writes, `Object.keys`, spread, and
 * `JSON.stringify` behave exactly as on the raw document (the methods are
 * not own properties, so serialisation is unaffected). A payload field
 * literally named `id` or `metadata` wins over the method.
 */
export function proxiedDocument<T extends Record<string, any>>(
  doc: Document<T>,
): ProxiedDocument<T> {
  return new Proxy(doc, {
    get(target, prop, receiver) {
      if (METHOD_NAMES.has(prop) && !(prop in target)) {
        if (prop === "id") return () => target._id_;
        return () => target._metadata_;
      }
      const value: unknown = Reflect.get(target, prop, receiver);
      return typeof value === "function" ? value.bind(target) : value;
    },
  }) as ProxiedDocument<T>;
}

/** Maps a page's documents through {@link proxiedDocument}. */
export function proxiedPage<T extends Record<string, any>>(
  page: Page<T>,
): Page<ProxiedDocument<T>> {
  return { ...page, data: page.data.map(proxiedDocument) };
}

const ENVELOPE_KEYS = new Set(["_id_", "_metadata_"]);

/**
 * Returns a shallow copy of a document payload without the server-managed
 * envelope keys (`_id_`, `_metadata_`). Only the server may assign those;
 * the client strips them from every outbound write so a caller can never
 * forge or clobber identity/metadata — including via a proxied document
 * round-tripped back into create/update. Query filters, schemas, and other
 * non-document payloads must NOT go through this helper.
 */
export function strippedData<T extends object>(data: T): T {
  if (!data || typeof data !== "object") return data;
  if (Array.isArray(data)) return [...data] as unknown as T;
  const out: Record<string, any> = {};
  for (const [key, value] of Object.entries(data)) {
    if (!ENVELOPE_KEYS.has(key)) out[key] = value;
  }
  return out as T;
}
