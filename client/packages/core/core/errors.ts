import type {
  ApiResponse,
} from "@asaidimu/network-client"
import { SystemError, Errors, type Issue } from "@asaidimu/utils-error"

interface ServerErrorBody {
  code?: string
  message?: string
  details?: { issues?: Issue[] }
}

export async function parseErrorBody(raw: Response): Promise<ServerErrorBody | null> {
  try {
    const body = await raw.clone().json() as { error?: ServerErrorBody }
    return body?.error ?? null
  } catch {
    return null
  }
}

// @note #err-20261103-4 issue : SDK drops server-sent error details
// @status open
// @priority medium
// @tags #sdk #errors
// @created 2026-11-03
// @author ai-code-reviewer
// @see #err-20261103-7
//
// The server serializes failures as { error: { code, message, details } }
// with details carrying issue.Cause for 4xx (transport_fasthttp.go
// writeError), but this parser only reads body.details?.issues - so
// validation causes and policy-denial details sent by the server never reach
// SDK callers. Align this shape with the wire format, or make the server
// emit details.issues.
export function toSystemError(response: ApiResponse<unknown>, body: ServerErrorBody | null): SystemError {
  if (body) {
    return new SystemError({
      code: body.code ?? "UNKNOWN",
      message: body.message ?? "Unknown error",
      issues: body.details?.issues,
    })
  }

  const rawMsg = response.error?.message
  if (rawMsg !== null && rawMsg !== undefined && typeof rawMsg === "object") {
    const obj = rawMsg as Record<string, unknown>
    return new SystemError({
      code: (obj.code as string) ?? "UNKNOWN",
      message: (obj.message as string) ?? "Unknown error",
    })
  }

  return new SystemError({
    code: "UNKNOWN",
    message: (rawMsg as string) ?? "Unknown error",
  })
}

export function notFound(path?: string): SystemError {
  return Errors.notFound(path)
}

export function permissionDenied(operation?: string): SystemError {
  return Errors.permissionDenied(operation)
}

export function internalError(cause?: unknown): SystemError {
  return Errors.internalError(cause)
}
