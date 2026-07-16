import type { QueryDSL } from "@asaidimu/query"
import { HestiaNetworkClient } from "../../core/client"
import { HestiaCollection } from "../../core/collection"
import type { Document, Page } from "../../core/types"
import type {
    PolicyOperation,
    PolicyRule,
    UpsertOperationRequest,
    UpsertRuleRequest,
} from "./types"

export class HestiaPolicies {
  private operations: HestiaCollection<PolicyOperation>
  private rules: HestiaCollection<PolicyRule>

  constructor(private client: HestiaNetworkClient) {
    this.operations = new HestiaCollection<PolicyOperation>(client, "_policy_operation_")
    this.rules = new HestiaCollection<PolicyRule>(client, "_policy_rule_")
  }

  // ── Operations ──

  async findOperations(query?: QueryDSL<PolicyOperation>): Promise<Page<PolicyOperation>> {
    return this.operations.find(query)
  }

  async listOperations(): Promise<Page<PolicyOperation>> {
    return this.operations.find({ pagination: { type: "offset", offset: 0, limit: 50 } })
  }

  async readOperation(name: string): Promise<Document<PolicyOperation> | undefined> {
    return this.operations.read(name)
  }

  async createOperation(data: UpsertOperationRequest): Promise<Document<PolicyOperation>> {
    return this.operations.create(data as any)
  }

  async updateOperation(name: string, data: UpsertOperationRequest): Promise<Document<PolicyOperation>> {
    return this.operations.update(name, data as any)
  }

  async deleteOperation(name: string): Promise<void> {
    return this.operations.delete(name)
  }

  // ── Rules ──

  async findRules(query?: QueryDSL<PolicyRule>): Promise<Page<PolicyRule>> {
    return this.rules.find(query)
  }

  async listRules(): Promise<Page<PolicyRule>> {
    return this.rules.find({ pagination: { type: "offset", offset: 0, limit: 50 } })
  }

  async readRule(name: string): Promise<Document<PolicyRule> | undefined> {
    return this.rules.read(name)
  }

  async createRule(data: UpsertRuleRequest): Promise<Document<PolicyRule>> {
    return this.rules.create(data as any)
  }

  async updateRule(name: string, data: UpsertRuleRequest): Promise<Document<PolicyRule>> {
    return this.rules.update(name, data as any)
  }

  async deleteRule(name: string): Promise<void> {
    return this.rules.delete(name)
  }


}
