import { HestiaNetworkClient } from "../../core/client"
import type { Document, Page, PaginationInfo, StoreEvent } from "../../core/types"
import type { DocumentStore } from "../../core/types"
import type {
  PolicyRule,
  CreateRuleRequest,
  UpdateRuleRequest,
  ValidateRuleRequest,
  ValidateRuleResult,
  ReloadResult,
} from "./types"

const RULES_PATH = "/system/policies/rule"

export class HestiaRules implements DocumentStore<PolicyRule, Record<string, unknown>, string, Record<string, unknown>, Record<string, unknown>, string, string, Record<string, unknown>> {
  constructor(private client: HestiaNetworkClient) {}

  async find(_query?: Record<string, unknown>): Promise<Page<PolicyRule>> {
    return this.list()
  }

  async list(_options?: Record<string, unknown>): Promise<Page<PolicyRule>> {
    const res = await this.client.get<{ data: { rules: PolicyRule[] } }>(RULES_PATH)
    const items = res.data?.data?.rules ?? []
    return {
      data: items.map(r => ({
        _id_: r.id,
        _metadata_: { checksum: "", created: "", updated: "", version: 1 },
        ...r,
      })),
      loading: false,
      page: { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
      error: null,
    }
  }

  async read(id: string): Promise<Document<PolicyRule> | undefined> {
    try {
      const res = await this.client.get<{ data: PolicyRule }>(
        `${RULES_PATH}/${encodeURIComponent(id)}`,
      )
      if (!res.data?.data) return undefined
      const r = res.data.data
      return {
        _id_: r.id,
        _metadata_: { checksum: "", created: "", updated: "", version: 1 },
        ...r,
      }
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND") return undefined
      throw err
    }
  }

  async create(props: { data: Partial<CreateRuleRequest>; options?: string }): Promise<Document<PolicyRule> | undefined> {
    const name = props.options ?? (props.data as any).name
    if (!name) throw new Error("Rule name is required for create")
    const res = await this.client.post<{ data: PolicyRule }>(
      `${RULES_PATH}/${encodeURIComponent(name)}`,
      props.data,
    )
    if (!res.data?.data) return undefined
    const r = res.data.data
    return {
      _id_: r.id,
      _metadata_: { checksum: "", created: "", updated: "", version: 1 },
      ...r,
    }
  }

  async update(props: { data: UpdateRuleRequest; options?: string }): Promise<Document<PolicyRule> | undefined> {
    const name = props.options!
    if (!name) throw new Error("Rule name is required for update")
    const res = await this.client.patch<{ data: PolicyRule }>(
      `${RULES_PATH}/${encodeURIComponent(name)}`,
      props.data,
    )
    if (!res.data?.data) return undefined
    const r = res.data.data
    return {
      _id_: r.id,
      _metadata_: { checksum: "", created: "", updated: "", version: 1 },
      ...r,
    }
  }

  async delete(name: string): Promise<void> {
    await this.client.delete(`${RULES_PATH}/${encodeURIComponent(name)}`)
  }

  async validate(request: ValidateRuleRequest): Promise<ValidateRuleResult> {
    const res = await this.client.check<{ data: ValidateRuleResult }>(
      RULES_PATH,
      request,
    )
    return res.data?.data ?? { valid: false }
  }

  async reload(): Promise<ReloadResult> {
    const res = await this.client.get<{ data: ReloadResult }>(RULES_PATH)
    return res.data!.data
  }

  async upload(_props: { file: File }): Promise<Document<PolicyRule> | undefined> {
    throw new Error("Upload not supported for rules")
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for rules")
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for rules")
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<PolicyRule>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for rules")
  }

  page(_options?: Record<string, unknown>): any {
    throw new Error("Pagination not supported for rules")
  }
}
