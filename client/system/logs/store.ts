import { type Transport } from "../../core/client"
import type { LogEntry, LogQuery, LogListResult } from "./types"

interface LogListEnvelope {
  data: LogListResult
}

interface StreamDoc {
  data: LogEntry
}

export class HestiaAppLogs {
  private apiPrefix: string

  constructor(
    private client: Transport,
    private baseUrl: string,
    apiPrefix: string = "/api",
  ) {
    this.apiPrefix = apiPrefix
  }

  async query(options?: LogQuery): Promise<LogListResult> {
    const res = await this.client.post<LogListEnvelope>(
      "/system/logs/list",
      options ?? {},
    )
    return res.data?.data ?? { entries: [], total: 0, has_more: false }
  }

  stream(options?: { level?: string }): {
    stream: () => AsyncIterable<LogEntry>
    cancel: () => void
    status: () => "active" | "cancelled" | "completed"
  } {
    const url = this.getStreamUrl(options)
    let eventSource: EventSource | null = null
    let currentStatus: "active" | "cancelled" | "completed" = "active"
    let pendingResolve: (() => void) | null = null

    const asyncStream = async function* () {
      const pending: LogEntry[] = []

      eventSource = new (EventSource as any)(url, { withCredentials: true })

      eventSource!.onmessage = (event) => {
        for (const line of event.data.split("\n")) {
          const trimmed = line.trim()
          if (!trimmed) continue
          try {
            const parsed = JSON.parse(trimmed) as StreamDoc
            if (parsed?.data) pending.push(parsed.data)
          } catch {
            // skip non-JSON lines
          }
        }
        if (pending.length > 0 && pendingResolve) {
          pendingResolve()
          pendingResolve = null
        }
      }

      eventSource!.onerror = () => {
        if (currentStatus === "active") currentStatus = "completed"
        if (pendingResolve) {
          pendingResolve()
          pendingResolve = null
        }
      }

      try {
        while (currentStatus === "active") {
          if (pending.length > 0) {
            yield pending.shift()!
          } else {
            await new Promise<void>((resolve) => {
              pendingResolve = resolve
              if (pending.length > 0) {
                resolve()
                pendingResolve = null
              }
            })
          }
        }
      } finally {
        eventSource?.close()
        if (pendingResolve) {
          pendingResolve()
          pendingResolve = null
        }
        if (currentStatus === "active") currentStatus = "completed"
      }
    }

    return {
      stream: () => asyncStream(),
      cancel: () => {
        if (currentStatus !== "active") return
        currentStatus = "cancelled"
        eventSource?.close()
      },
      status: () => currentStatus,
    }
  }

  private getStreamUrl(options?: { level?: string }): string {
    const params = new URLSearchParams()
    if (options?.level) params.set("level", options.level)
    const qs = params.toString()
    return `${this.baseUrl}${this.apiPrefix}/system/logs/stream${qs ? "?" + qs : ""}`
  }
}
