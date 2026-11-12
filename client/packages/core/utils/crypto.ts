// sha256.ts
//
// SHA-256 hashing with a Web Crypto fast path and a pure-JS fallback for
// non-secure contexts (HTTP + non-localhost). The fallback is a faithful
// implementation of FIPS 180-4, optimized for the small-to-medium chunks
// produced by ResumableUpload (typically ≤ 8 MiB).

const K = new Uint32Array([
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1,
  0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
  0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786,
  0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147,
  0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
  0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
  0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a,
  0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
  0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
]);

const HEX_TABLE: string[] = new Array(256);
for (let i = 0; i < 256; i++) HEX_TABLE[i] = i.toString(16).padStart(2, "0");

/**
 * Dev-only escape hatch for non-secure contexts (HTTP + non-localhost).
 *
 * Set `globalThis.__HESTIA_ALLOW_INSECURE_CRYPTO__ = true` (or on `window`)
 * before creating the blob client to permit a pure-JS SHA-256 fallback when
 * Web Crypto is unavailable. Production builds should always serve over HTTPS
 * so that `crypto.subtle` is present.
 */
function allowInsecureCrypto(): boolean {
  try {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const g = globalThis as any;
    return !!(
      g.__HESTIA_ALLOW_INSECURE_CRYPTO__ ||
      g.window?.__HESTIA_ALLOW_INSECURE_CRYPTO__
    );
  } catch {
    return false;
  }
}

/**
 * Pure-JS SHA-256 (fallback only). Correct for any `Uint8Array`-sized input.
 *
 * Optimization notes vs. the previous implementation:
 *  - Full 64-byte blocks are hashed directly from the input buffer; only the
 *    final ≤127-byte tail is copied into a small padding buffer. This avoids
 *    allocating + copying the entire input.
 *  - Word loads use explicit byte shifts instead of DataView.getUint32,
 *    which is measurably faster in V8 for this access pattern.
 *  - Hex output uses a 256-entry lookup table.
 *  - The 64-bit message length is computed correctly (previous version
 *    forced the high word to 0).
 */
function sha256HexPure(data: ArrayBuffer | ArrayBufferView): string {
  const bytes =
    data instanceof ArrayBuffer
      ? new Uint8Array(data)
      : new Uint8Array(
          (data as ArrayBufferView).buffer,
          (data as ArrayBufferView).byteOffset,
          (data as ArrayBufferView).byteLength,
        );

  const len = bytes.length;
  // bit length = len * 8, split into high/low 32-bit halves
  const bitLenLo = (len * 8) >>> 0;
  const bitLenHi = Math.floor(len / 0x20000000) >>> 0; // len / 2^29

  let h0 = 0x6a09e667 | 0;
  let h1 = 0xbb67ae85 | 0;
  let h2 = 0x3c6ef372 | 0;
  let h3 = 0xa54ff53a | 0;
  let h4 = 0x510e527f | 0;
  let h5 = 0x9b05688c | 0;
  let h6 = 0x1f83d9ab | 0;
  let h7 = 0x5be0cd19 | 0;

  const w = new Uint32Array(64);

  const rotr = (x: number, n: number) => (x >>> n) | (x << (32 - n));

  function processBlock(block: Uint8Array, off: number): void {
    // Load 16 big-endian words directly from the source bytes.
    for (let t = 0; t < 16; t++) {
      const o = off + (t << 2);
      w[t] =
        (block[o]! << 24) |
        (block[o + 1]! << 16) |
        (block[o + 2]! << 8) |
        block[o + 3]!;
    }
    // Extend the message schedule.
    for (let t = 16; t < 64; t++) {
      const x = w[t - 15]!;
      const y = w[t - 2]!;
      const s0 = rotr(x, 7) ^ rotr(x, 18) ^ (x >>> 3);
      const s1 = rotr(y, 17) ^ rotr(y, 19) ^ (y >>> 10);
      w[t] = (w[t - 16]! + s0 + w[t - 7]! + s1) | 0;
    }

    let a = h0,
      b = h1,
      c = h2,
      d = h3;
    let e = h4,
      f = h5,
      g = h6,
      h = h7;

    for (let t = 0; t < 64; t++) {
      const S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
      const ch = (e & f) ^ (~e & g);
      const t1 = (h + S1 + ch + K[t]! + w[t]!) | 0;
      const S0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
      const maj = (a & b) ^ (a & c) ^ (b & c);
      const t2 = (S0 + maj) | 0;

      h = g;
      g = f;
      f = e;
      e = (d + t1) | 0;
      d = c;
      c = b;
      b = a;
      a = (t1 + t2) | 0;
    }

    h0 = (h0 + a) | 0;
    h1 = (h1 + b) | 0;
    h2 = (h2 + c) | 0;
    h3 = (h3 + d) | 0;
    h4 = (h4 + e) | 0;
    h5 = (h5 + f) | 0;
    h6 = (h6 + g) | 0;
    h7 = (h7 + h) | 0;
  }

  // 1. Hash every complete 64-byte block directly from the input.
  const fullBlocks = len >>> 6;
  for (let b = 0; b < fullBlocks; b++) processBlock(bytes, b << 6);

  // 2. Build a small padding tail (≤ 128 bytes) and hash it.
  const tailStart = fullBlocks << 6;
  const tailLen = len - tailStart;
  const padBlockSize = tailLen <= 55 ? 64 : 128;
  const pad = new Uint8Array(padBlockSize);
  pad.set(bytes.subarray(tailStart));
  pad[tailLen] = 0x80;
  const pv = new DataView(pad.buffer, pad.byteOffset, pad.byteLength);
  pv.setUint32(padBlockSize - 8, bitLenHi, false);
  pv.setUint32(padBlockSize - 4, bitLenLo, false);

  processBlock(pad, 0);
  if (padBlockSize === 128) processBlock(pad, 64);

  // 3. Serialize the state as 32 big-endian bytes.
  const out = new Uint8Array(32);
  const ov = new DataView(out.buffer);
  ov.setUint32(0, h0, false);
  ov.setUint32(4, h1, false);
  ov.setUint32(8, h2, false);
  ov.setUint32(12, h3, false);
  ov.setUint32(16, h4, false);
  ov.setUint32(20, h5, false);
  ov.setUint32(24, h6, false);
  ov.setUint32(28, h7, false);

  let hex = "";
  for (let i = 0; i < 32; i++) hex += HEX_TABLE[out[i]!];
  return hex;
}

export async function sha256Hex(
  buffer: ArrayBuffer | ArrayBufferView,
): Promise<string> {
  const subtle = typeof crypto !== "undefined" ? crypto.subtle : undefined;
  if (subtle && typeof subtle.digest === "function") {
    // @ts-expect-error
    const digest = await subtle.digest("SHA-256", buffer as BufferSource);
    const bytes = new Uint8Array(digest);
    let hex = "";
    for (let i = 0; i < bytes.length; i++) hex += HEX_TABLE[bytes[i]!];
    return hex;
  }

  if (allowInsecureCrypto()) {
    return sha256HexPure(buffer);
  }

  throw new Error(
    "Web Crypto (crypto.subtle) is unavailable. File uploads require a secure context (HTTPS or localhost). " +
      "For local development over HTTP you may set globalThis.__HESTIA_ALLOW_INSECURE_CRYPTO__ = true " +
      "to enable a pure-JS fallback (not recommended for production).",
  );
}
