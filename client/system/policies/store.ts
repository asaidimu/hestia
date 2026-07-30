import type { QueryDSL } from "@asaidimu/query"
import { type Transport } from "../../core/client"
import { ReactiveDataStore } from "@asaidimu/utils-store"
import { createPagedController } from "../../core/pager"
import type { Document, Page, PagedData, PaginationInfo, StoreEvent } from "../../core/types"
import type { DocumentStore } from "../../core/types"
import type {
    Policy,
    CreatePolicyRequest,
    UpdatePolicyRuleRequest,
    SetPolicyEnabledRequest,
    UpdatePolicyRequest,
} from "./types"

const POLICY_COLLECTION = "_operation_policy_"

function rawToPolicy(doc: any): Policy {
    return {
        id: doc._id_ ?? doc.id ?? "",
        operationName: doc.operation ?? doc.operationName ?? "",
        ruleName: doc.rule ?? doc.ruleName ?? "",
        enabled: doc.enabled ?? true,
        protected: doc.protected ?? false,
        rateLimit: doc.rateLimit ?? undefined,
        throttle: doc.throttle ?? undefined,
    }
}

export class HestiaPolicies implements DocumentStore<Policy, QueryDSL<Policy>, string, QueryDSL<Policy>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pager: PagedData<Policy>

  constructor(private client: Transport) {
    this.pager = createPagedController<Policy>(
      "policies",
      new ReactiveDataStore<any>({}),
      {},
      (query) => this.find(query),
    )
  }

  async find(_query?: QueryDSL<Policy>): Promise<Page<Policy>> {
    return this.query(_query ?? {})
  }

  async query(qdsl: Record<string, unknown>): Promise<Page<Policy>> {
    const res = await this.client.dispatch<{ data: any[]; metadata?: { page?: PaginationInfo } }>(
      "system:collections:document:query",
      { arguments: { name: POLICY_COLLECTION }, payload: qdsl },
    )
    const items = res.data?.data ?? []
    const pagination = res.data?.metadata?.page
    const policies: Document<Policy>[] = items.map((doc: any) => {
      const p = rawToPolicy(doc)
      return { data: p, _id_: doc._id_, _metadata_: doc._metadata_ }
    })
    return {
      data: policies,
      loading: false,
      page: pagination ?? { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
      error: null,
    }
  }

  async list(_options?: QueryDSL<Policy>): Promise<Page<Policy>> {
    const res = await this.client.dispatch<{ data: { policies: Policy[] } }>(
      "system:policies:policy:list",
    )
    const items = res.data?.data?.policies ?? []
    return {
      data: items.map(p => ({ data: p, metadata: {} })),
      loading: false,
      page: { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
      error: null,
    }
  }

  async read(id: string): Promise<Document<Policy> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: Policy }>(
        "system:policies:policy:list",
      )
      const items = (res.data?.data as any)?.policies ?? []
      const match = items.find((p: any) => p._id_ === id || p.operation === id || p.id === id)
      if (!match) return undefined
      return { data: rawToPolicy(match), metadata: {} }
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND") return undefined
      throw err
    }
  }

  async create(props: { data: Partial<CreatePolicyRequest>; options?: string }): Promise<Document<Policy> | undefined> {
    const name = props.options ?? (props.data as any).name
    if (!name) throw new Error("Operation name is required for create")
    const body: CreatePolicyRequest = {
      ruleName: (props.data as any).ruleName ?? "",
      rateLimit: (props.data as any).rateLimit,
      throttle: (props.data as any).throttle,
    }
    const res = await this.client.dispatch<{ data: Policy }>(
      "system:policies:policy:create",
      { arguments: { name }, payload: body },
    )
    if (!res.data?.data) return undefined
    return { data: res.data.data, metadata: {} }
  }

  async update(props: { data: UpdatePolicyRequest; options?: string }): Promise<Document<Policy> | undefined> {
    const name = props.options!
    if (!name) throw new Error("Operation name is required for update")
    const payload: Record<string, unknown> = {}
    if (props.data.ruleName !== undefined) payload.ruleName = props.data.ruleName
    if (props.data.enabled !== undefined) payload.enabled = props.data.enabled
    if (props.data.rateLimit !== undefined) payload.rateLimit = props.data.rateLimit
    if (props.data.throttle !== undefined) payload.throttle = props.data.throttle
    const res = await this.client.dispatch<{ data: Policy }>(
      "system:policies:policy:update",
      { arguments: { name }, payload },
    )
    if (!res.data?.data) return undefined
    return { data: res.data.data, metadata: {} }
  }

  async setEnabled(name: string, enabled: boolean): Promise<Document<Policy>> {
    const res = await this.client.dispatch<{ data: Policy }>(
      "system:policies:policy:update",
      { arguments: { name }, payload: { enabled } as SetPolicyEnabledRequest },
    )
    return { data: res.data!.data, metadata: {} }
  }

  async delete(_id: string): Promise<void> {
    throw new Error("Policies cannot be deleted; disable instead")
  }

  async upload(_props: { file: File }): Promise<Document<Policy> | undefined> {
    throw new Error("Upload not supported for policies")
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for policies")
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for policies")
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<Policy>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for policies")
  }

  page(_options?: Record<string, unknown>): PagedData<Policy> {
    return this.pager
  }
}
