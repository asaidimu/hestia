import type { QueryDSL } from "@asaidimu/query"
import { HestiaNetworkClient } from "../../core/client"
import { ReactiveDataStore } from "@asaidimu/utils-store"
import { createPagedController } from "../../core/pager"
import type { Document, Page, PagedData, PaginationInfo, StoreEvent } from "../../core/types"
import type { DocumentStore } from "../../core/types"
import type {
    OperationPolicy,
    IamRule,
    UpsertOperationRequest,
    UpsertRuleRequest,
    ReloadResult,
} from "./types"

const OPERATIONS_COLLECTION = "_operation_policy_"
const RULES_COLLECTION = "_iam_rule_"

export class HestiaPolicies implements DocumentStore<OperationPolicy, QueryDSL<OperationPolicy>, string, QueryDSL<OperationPolicy>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pager: PagedData<OperationPolicy>
  private operationsPath = "/system/policies/operation"
  private rulesPath = "/system/policies/rule"

  constructor(private client: HestiaNetworkClient) {
    this.pager = createPagedController<OperationPolicy>(
      "policies",
      new ReactiveDataStore<any>({}),
      {},
      (query) => this.find(query),
    )
  }

  private async collectionQuery<T extends Record<string, any>>(
    collection: string,
    query?: Record<string, unknown>,
  ): Promise<Page<T>> {
    const res = await this.client.post<{
      data: Document<T>[];
      metadata?: { page?: PaginationInfo };
    }>(`/system/collections/document/${encodeURIComponent(collection)}/query`, query ?? {})
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

  async find(query?: QueryDSL<OperationPolicy>): Promise<Page<OperationPolicy>> {
    return this.collectionQuery<OperationPolicy>(OPERATIONS_COLLECTION, query as Record<string, unknown> | undefined)
  }

  async list(options?: QueryDSL<OperationPolicy>): Promise<Page<OperationPolicy>> {
    return this.find(options ?? { pagination: { type: "offset", offset: 0, limit: 50 } })
  }

  async read(id: string): Promise<Document<OperationPolicy> | undefined> {
    try {
      const res = await this.client.get<{ data: Document<OperationPolicy> }>(
        `${this.operationsPath}/${encodeURIComponent(id)}`,
      )
      return res.data?.data
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND" || err?.code === "INTERNAL_ERROR" && typeof err?.message === "string" && err.message.includes("not found")) return undefined
      throw err
    }
  }

  async create(props: { data: Partial<OperationPolicy> }): Promise<Document<OperationPolicy> | undefined> {
    const name = props.data.name
    if (!name) throw new Error("Operation name is required for create")
    const res = await this.client.patch<{ data: Document<OperationPolicy> }>(
      `${this.operationsPath}/${encodeURIComponent(name)}`,
      props.data as UpsertOperationRequest,
    )
    return res.data!.data
  }

  async update(props: { data: Partial<OperationPolicy>; options?: string }): Promise<Document<OperationPolicy> | undefined> {
    const name = props.options!
    if (!name) throw new Error("Operation name is required for update")
    const res = await this.client.patch<{ data: Document<OperationPolicy> }>(
      `${this.operationsPath}/${encodeURIComponent(name)}`,
      props.data as UpsertOperationRequest,
    )
    return res.data!.data
  }

  async delete(id: string): Promise<void> {
    await this.client.delete(`${this.operationsPath}/${encodeURIComponent(id)}`)
  }

  async upsertOperation(
    name: string,
    data: UpsertOperationRequest,
  ): Promise<Document<OperationPolicy>> {
    const res = await this.client.patch<{ data: Document<OperationPolicy> }>(
      `${this.operationsPath}/${encodeURIComponent(name)}`,
      data,
    )
    return res.data!.data
  }

  async getRule(name: string): Promise<Document<IamRule> | undefined> {
    try {
      const res = await this.client.get<{ data: Document<IamRule> }>(
        `${this.rulesPath}/${encodeURIComponent(name)}`,
      )
      return res.data?.data
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND" || err?.code === "INTERNAL_ERROR" && typeof err?.message === "string" && err.message.includes("not found")) return undefined
      throw err
    }
  }

  async upsertRule(
    name: string,
    data: UpsertRuleRequest,
  ): Promise<Document<IamRule>> {
    const res = await this.client.patch<{ data: Document<IamRule> }>(
      `${this.rulesPath}/${encodeURIComponent(name)}`,
      data,
    )
    return res.data!.data
  }

  async deleteRule(name: string): Promise<void> {
    await this.client.delete(`${this.rulesPath}/${encodeURIComponent(name)}`)
  }

  async listRules(options?: Record<string, unknown>): Promise<Page<IamRule>> {
    return this.collectionQuery<IamRule>(RULES_COLLECTION, options)
  }

  async validateRule(
    expression: string,
  ): Promise<boolean> {
    const res = await this.client.post<{ data: { valid: boolean } }>(
      `${this.rulesPath}/validate`,
      { expression },
    )
    return res.data?.data?.valid ?? false
  }

  async reload(): Promise<ReloadResult> {
    const res = await this.client.get<{ data: ReloadResult }>(
      `${this.rulesPath}/reload`,
    )
    return res.data!.data
  }

  async upload(_props: { file: File }): Promise<Document<OperationPolicy> | undefined> {
    throw new Error("Upload not supported for policies")
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for policies")
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for policies")
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<OperationPolicy>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for policies")
  }

  page(_options?: Record<string, unknown>): PagedData<OperationPolicy> {
    return this.pager
  }
}
