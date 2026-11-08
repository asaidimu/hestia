import type { QueryDSL } from "@asaidimu/query"
import { ReactiveDataStore } from "@asaidimu/utils-store"
import { createAsyncStream } from "../../core/async-stream"
import { type Transport } from "../../core/client"
import { createPagedController, type PageOptions } from "../../core/pager"
import type { Document, Page, PagedData, PaginationInfo, StoreEvent } from "../../core/types"
import type { DocumentStore } from "../../core/types"
import type { AuditEntry } from "./types"

interface FindEnvelope {
  data: Document<AuditEntry>[]
  metadata?: { page?: PaginationInfo }
}

interface StreamDoc {
  data: Document<AuditEntry>
}

export class HestiaAuditLogs implements DocumentStore<AuditEntry, QueryDSL<AuditEntry>, string, QueryDSL<AuditEntry>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pagerOptions: PageOptions<AuditEntry> = {}
  private pager: PagedData<AuditEntry>

  private apiPrefix: string;

  constructor(
    private client: Transport,
    private baseUrl: string,
    apiPrefix: string = "/api",
  ) {
    this.apiPrefix = apiPrefix;
    this.pager = createPagedController<AuditEntry>(
      "_audit_log_",
      new ReactiveDataStore<any>({}),
      this.pagerOptions,
      (query) => this.find(query),
    )
  }

  async find(query?: QueryDSL<AuditEntry>): Promise<Page<AuditEntry>> {
    const res = await this.client.post<FindEnvelope>(
      "/system/collections/audit_log/query",
      query ?? {},
    )

    const items = res.data?.data ?? []
    const pageMeta = res.data?.metadata?.page ?? {
      number: 1,
      size: items.length,
      count: items.length,
      total: items.length,
      pages: 1,
    }

    return { data: items, loading: false, page: pageMeta, error: null }
  }

  async list(options?: QueryDSL<AuditEntry>): Promise<Page<AuditEntry>> {
    return this.find(
      options ?? { pagination: { type: "offset", offset: 0, limit: 50 } },
    )
  }

  async read(_id: string): Promise<Document<AuditEntry> | undefined> {
    throw new Error("Read by ID not supported for audit logs; use find with filters")
  }

  async create(_props: { data: Partial<AuditEntry> }): Promise<Document<AuditEntry> | undefined> {
    throw new Error("Audit logs are write-only; entries are created by the system")
  }

  async update(_props: { data: Partial<AuditEntry>; id?: string }): Promise<Document<AuditEntry> | undefined> {
    throw new Error("Audit logs are append-only; updates are not allowed")
  }

  async delete(_id: string): Promise<void> {
    throw new Error("Audit logs are append-only; deletion is not allowed")
  }

  async upload(_props: { file: File }): Promise<Document<AuditEntry> | undefined> {
    throw new Error("Upload not supported for audit logs")
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Use stream() for real-time audit log entries")
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for audit logs")
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<AuditEntry>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    const source = createAsyncStream<Document<AuditEntry>>()
    const controller = new AbortController()
    let started = false

    const start = () => {
      if (started) return
      started = true
      if (source.status() !== "active") return

      this.client
        .openStream(
          "/system/audit/log/stream",
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

  page(_options?: Record<string, unknown>): PagedData<AuditEntry> {
    return this.pager
  }

  getStreamUrl(): string {
    return `${this.baseUrl}${this.apiPrefix}/system/audit/log/stream`
  }
}
