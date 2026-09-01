/**
 * Server-side error factories.
 *
 * The mock throws real `SystemError` instances (from `@asaidimu/utils-error`)
 * so the Hestia client's catch-sites — which inspect `err.code` — behave
 * identically against the mock and a real server.
 */

import { SystemError } from "@asaidimu/utils-error";

export type MockErrorCode =
  | "SYNC-001-NF"       // not found (404)
  | "SYNC-002-DUP"      // duplicate key (409)
  | "SYNC-003-VC"       // version conflict (409)
  | "VAL-001"           // validation failed (400)
  | "VAL-002"           // required field missing (400)
  | "AUTH-001-DENIED"   // permission denied (403)
  | "AUTH-002-UNAUTH"   // unauthenticated (401)
  | "AUTH-003-CRED"     // invalid credentials (401)
  | "NOT_FOUND"         // blob-store style not found (404)
  | "NO_ACTIVE_SESSION" // logout with no session (404)
  | "ROUTE_NOT_FOUND"   // unknown route (404)
  | "NOT_IMPLEMENTED"   // not implemented (501)
  | "RESTART_REQUIRED"  // restart needed (503)
  | "INTERNAL_ERROR";   // internal error (500)

const STATUS: Record<MockErrorCode, number> = {
  "SYNC-001-NF": 404,
  "SYNC-002-DUP": 409,
  "SYNC-003-VC": 409,
  "VAL-001": 400,
  "VAL-002": 400,
  "AUTH-001-DENIED": 403,
  "AUTH-002-UNAUTH": 401,
  "AUTH-003-CRED": 401,
  NOT_FOUND: 404,
  NO_ACTIVE_SESSION: 404,
  ROUTE_NOT_FOUND: 404,
  NOT_IMPLEMENTED: 501,
  RESTART_REQUIRED: 503,
  INTERNAL_ERROR: 500,
};

/** HTTP status the mock would have answered with for this error code. */
export function statusForError(err: unknown): number {
  if (err && typeof err === "object" && "code" in err) {
    const code = (err as { code: string }).code as MockErrorCode;
    return STATUS[code] ?? 500;
  }
  return 500;
}

export class MockHttpError extends SystemError {
  /** HTTP status the mock transport associates with this error. */
  public readonly httpStatus: number;

  constructor(code: MockErrorCode, message: string, issues?: unknown[]) {
    super({ code, message, issues: issues as never });
    this.name = "SystemError";
    this.httpStatus = STATUS[code] ?? 500;
  }
}

export const err = {
  notFound: (what: string) => new MockHttpError("SYNC-001-NF", `${what} not found`),
  /** Blob-store style not found (the SDK's blob client checks `NOT_FOUND`). */
  blobNotFound: (what: string) => new MockHttpError("NOT_FOUND", `${what} not found`),
  duplicate: (what: string) => new MockHttpError("SYNC-002-DUP", `${what} already exists`),
  conflict: (message: string) => new MockHttpError("SYNC-003-VC", message),
  validation: (message: string, issues?: unknown[]) => new MockHttpError("VAL-001", message, issues),
  required: (field: string) => new MockHttpError("VAL-002", `Required field missing: ${field}`),
  denied: (operation = "operation") =>
    new MockHttpError("AUTH-001-DENIED", `You do not have permission to perform ${operation}`),
  unauthenticated: () =>
    new MockHttpError("AUTH-002-UNAUTH", "This operation requires an authenticated session"),
  invalidCredentials: () => new MockHttpError("AUTH-003-CRED", "invalid email or password"),
  noActiveSession: () => new MockHttpError("NO_ACTIVE_SESSION", "no active session to revoke"),
  routeNotFound: (name: string) => new MockHttpError("ROUTE_NOT_FOUND", `No registered route for handler: ${name}`),
  notImplemented: (what: string) => new MockHttpError("NOT_IMPLEMENTED", `${what} is not implemented by the mock server`),
  restartRequired: (message = "the change requires a server restart") =>
    new MockHttpError("RESTART_REQUIRED", message),
  internal: (message: string, cause?: unknown) => {
    const error = new MockHttpError("INTERNAL_ERROR", cause ? `${message}: ${String(cause)}` : message);
    return error;
  },
};
