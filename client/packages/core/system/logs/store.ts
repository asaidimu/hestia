import { createAsyncStream } from "../../core/async-stream"
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
    const source = createAsyncStream<LogEntry>()
    const controller = new AbortController()
    const params = new URLSearchParams()
    if (options?.level) params.set("level", options.level)
    const query = params.toString()
    const path = `/system/logs/stream${query ? "?" + query : ""}`
    let started = false

    const start = () => {
      if (started) return
      started = true
      if (source.status() !== "active") return

      this.client
        .openStream(
          path,
          {
            onMessage(data) {
              try {
                const parsed = JSON.parse(data) as StreamDoc
                if (parsed?.data) source.push(parsed.data)
              } catch {
                // skip non-JSON lines
              }
            },
            onError: () => source.end(),
            onClose: () => source.end(),
          },
          { signal: controller.signal },
        )
        .catch(() => source.end())
    }

    const iterate = async function* () {
      start()
      yield* source.stream()
    }

    return {
      stream: () => iterate(),
      cancel: () => {
        controller.abort()
        source.cancel()
      },
      status: source.status,
    }
  }

  private getStreamUrl(options?: { level?: string }): string {
    const params = new URLSearchParams()
    if (options?.level) params.set("level", options.level)
    const qs = params.toString()
    return `${this.baseUrl}${this.apiPrefix}/system/logs/stream${qs ? "?" + qs : ""}`
  }
}
