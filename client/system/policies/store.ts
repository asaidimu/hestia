import type { QueryDSL } from "@asaidimu/query"
import { HestiaNetworkClient } from "../../core/client"
import { ReactiveDataStore } from "@asaidimu/utils-store"
import { createPagedController } from "../../core/pager"
import type { Document, Page, PagedData, PaginationInfo, StoreEvent } from "../../core/types"
import type { DocumentStore } from "../../core/types"
import type {
    Policy,
    CreatePolicyRequest,
    UpdatePolicyRuleRequest,
    SetPolicyEnabledRequest,
} from "./types"

const POLICIES_PATH = "/system/policies/policy"
const POLICY_COLLECTION = "_operation_policy_"

export class HestiaPolicies implements DocumentStore<Policy, QueryDSL<Policy>, string, QueryDSL<Policy>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pager: PagedData<Policy>

  constructor(private client: HestiaNetworkClient) {
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
    const res = await this.client.post<{ data: any[]; metadata?: { page?: PaginationInfo } }>(
      `/system/collections/document/${POLICY_COLLECTION}/query`,
      qdsl,
    )
    const items = res.data?.data ?? []
    const pagination = res.data?.metadata?.page
    const policies: Document<Policy>[] = items.map((doc: any) => ({
      _id_: doc._id_,
      _metadata_: doc._metadata_,
      id: doc._id_,
      operationName: doc.operation ?? "",
      ruleName: doc.rule ?? "",
      enabled: doc.enabled ?? true,
      protected: doc.protected ?? false,
    }))
    return {
      data: policies,
      loading: false,
      page: pagination ?? { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
      error: null,
    }
  }

  async list(_options?: QueryDSL<Policy>): Promise<Page<Policy>> {
    const res = await this.client.get<{ data: { policies: Policy[] } }>(POLICIES_PATH)
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
      const res = await this.client.get<{ data: Policy }>(
        `${POLICIES_PATH}/${encodeURIComponent(id)}`,
      )
      if (!res.data?.data) return undefined
      return { data: res.data.data, metadata: {} }
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND") return undefined
      throw err
    }
  }

  async create(props: { data: Partial<CreatePolicyRequest>; options?: string }): Promise<Document<Policy> | undefined> {
    const name = props.options ?? (props.data as any).name
    if (!name) throw new Error("Operation name is required for create")
    const body: CreatePolicyRequest = { ruleName: (props.data as any).ruleName ?? "" }
    const res = await this.client.post<{ data: Policy }>(
      `${POLICIES_PATH}/${encodeURIComponent(name)}`,
      body,
    )
    if (!res.data?.data) return undefined
    return { data: res.data.data, metadata: {} }
  }

  async update(props: { data: UpdatePolicyRuleRequest; options?: string }): Promise<Document<Policy> | undefined> {
    const name = props.options!
    if (!name) throw new Error("Operation name is required for update")
    const res = await this.client.patch<{ data: Policy }>(
      `${POLICIES_PATH}/${encodeURIComponent(name)}`,
      props.data,
    )
    if (!res.data?.data) return undefined
    return { data: res.data.data, metadata: {} }
  }

  async setEnabled(name: string, enabled: boolean): Promise<Document<Policy>> {
    const res = await this.client.patch<{ data: Policy }>(
      `${POLICIES_PATH}/${encodeURIComponent(name)}`,
      { enabled } as SetPolicyEnabledRequest,
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
