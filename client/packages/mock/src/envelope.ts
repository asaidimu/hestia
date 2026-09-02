/**
 * Response envelope helpers mirroring the Go HTTP transport
 * (`{data: ..., metadata: {timestamp, request, page?}}`).
 */

export interface PaginationInfo {
  number: number;
  size: number;
  count: number;
  total: number;
  pages: number;
}

export interface Envelope<T> {
  data: T;
  metadata?: {
    timestamp: string;
    request: string;
    page?: PaginationInfo;
    [key: string]: unknown;
  };
}

export interface ServerResponse {
  status: number;
  body: Envelope<unknown> | undefined;
}

export function ok(
  data: unknown,
  opts?: { page?: PaginationInfo; request?: string; status?: number },
): ServerResponse {
  return {
    status: opts?.status ?? 200,
    body: {
      data,
      metadata: {
        timestamp: new Date().toISOString(),
        request: opts?.request ?? "",
        ...(opts?.page ? { page: opts.page } : {}),
      },
    },
  };
}

export function noContent(): ServerResponse {
  return { status: 204, body: undefined };
}

/** Compute pagination metadata for a slice of a larger result set. */
export function pageMeta(total: number, offset: number, limit: number, count: number): PaginationInfo {
  const size = Math.max(1, limit);
  return {
    number: Math.floor(Math.max(0, offset) / size) + 1,
    size,
    count,
    total,
    pages: Math.max(1, Math.ceil(total / size)),
  };
}
