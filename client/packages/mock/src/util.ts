/** Shared runtime utilities: ids, hashing, time, bytes. */

/** RFC4122 v4 UUID (crypto-backed with a deterministic fallback). */
export function uuid(): string {
  const c = globalThis.crypto;
  if (c && "randomUUID" in c) return c.randomUUID();
  // Fallback for exotic environments.
  const bytes = new Uint8Array(16);
  for (let i = 0; i < 16; i++) bytes[i] = Math.floor(Math.random() * 256);
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

export function randomHex(bytes: number): string {
  const buf = new Uint8Array(bytes);
  globalThis.crypto.getRandomValues(buf);
  return [...buf].map((b) => b.toString(16).padStart(2, "0")).join("");
}

export function randomToken(prefix = "hst"): string {
  return `${prefix}_${randomHex(24)}`;
}

/** SHA-256 hex digest of a string or byte sequence. */
export async function sha256Hex(input: string | Uint8Array): Promise<string> {
  const data =
    typeof input === "string"
      ? new TextEncoder().encode(input)
      : input;
  const digest = await globalThis.crypto.subtle.digest("SHA-256", data as BufferSource);
  return [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

/** Salted password hash — `sha256(salt:password)` stored as `salt:hash`. */
export async function hashPassword(password: string): Promise<string> {
  const salt = randomHex(12);
  const hash = await sha256Hex(`${salt}:${password}`);
  return `${salt}:${hash}`;
}

export async function verifyPassword(password: string, stored: string): Promise<boolean> {
  const [salt, hash] = stored.split(":");
  if (!salt || !hash) return false;
  return (await sha256Hex(`${salt}:${password}`)) === hash;
}

/** Deterministic JSON stringify (stable key order) for checksums. */
export function stableStringify(value: unknown): string {
  if (value === null || typeof value !== "object") return JSON.stringify(value) ?? "null";
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(",")}]`;
  const entries = Object.entries(value as Record<string, unknown>)
    .filter(([, v]) => v !== undefined)
    .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
    .map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`);
  return `{${entries.join(",")}}`;
}

export function nowIso(): string {
  return new Date().toISOString();
}

export function sleep(ms: number): Promise<void> {
  if (!ms || ms <= 0) return Promise.resolve();
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** Convert any binary payload (Blob/File/ArrayBuffer/TypedArray) to bytes. */
export async function toBytes(payload: unknown): Promise<Uint8Array> {
  if (payload == null) return new Uint8Array(0);
  if (payload instanceof Uint8Array) return payload;
  if (payload instanceof ArrayBuffer) return new Uint8Array(payload);
  if (typeof Blob !== "undefined" && payload instanceof Blob) {
    return new Uint8Array(await payload.arrayBuffer());
  }
  if (ArrayBuffer.isView(payload)) {
    return new Uint8Array(payload.buffer, payload.byteOffset, payload.byteLength);
  }
  if (typeof payload === "string") return new TextEncoder().encode(payload);
  throw new Error(`Unsupported binary payload type: ${typeof payload}`);
}

/** True when the value looks like a binary body (Blob/File/ArrayBuffer/TypedArray). */
export function isBinary(payload: unknown): boolean {
  return (
    payload instanceof ArrayBuffer ||
    ArrayBuffer.isView(payload) ||
    (typeof Blob !== "undefined" && payload instanceof Blob) ||
    (typeof File !== "undefined" && payload instanceof File)
  );
}

/** Safe structural clone for values returned to callers. */
export function clone<T>(value: T): T {
  return structuredClone(value);
}

/** Compare two values with numeric coercion when both sides are numeric-like. */
export function compareValues(a: unknown, b: unknown): number {
  if (a === b) return 0;
  if (a == null) return -1;
  if (b == null) return 1;

  const na = toNumber(a);
  const nb = toNumber(b);
  if (na !== null && nb !== null) return na === nb ? 0 : na < nb ? -1 : 1;

  const sa = String(a);
  const sb = String(b);
  return sa === sb ? 0 : sa < sb ? -1 : 1;
}

function toNumber(v: unknown): number | null {
  if (typeof v === "number") return Number.isFinite(v) ? v : null;
  if (typeof v === "boolean") return v ? 1 : 0;
  if (typeof v === "string" && v.trim() !== "") {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
    const d = Date.parse(v);
    if (!Number.isNaN(d)) return d;
  }
  if (v instanceof Date) return v.getTime();
  return null;
}

/**
 * Minimal deep equality used by `eq`/`neq`/`in` filters — primitives by
 * value, arrays element-wise, plain objects key-wise.
 */
export function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (a == null || b == null) return a === b;
  if (typeof a !== "object" || typeof b !== "object") return a === b;
  if (Array.isArray(a) !== Array.isArray(b)) return false;
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false;
    return a.every((v, i) => deepEqual(v, b[i]));
  }
  const ka = Object.keys(a as object);
  const kb = Object.keys(b as object);
  if (ka.length !== kb.length) return false;
  return ka.every((k) =>
    deepEqual((a as Record<string, unknown>)[k], (b as Record<string, unknown>)[k]),
  );
}
