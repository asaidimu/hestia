import { HestiaNetworkClient } from "../../core/client"
import type { Document, Page } from "../../core/types"
import type { Operation } from "./types"

const OPERATIONS_PATH = "/system/policies/operation"

export class HestiaOperations {
  constructor(private client: HestiaNetworkClient) {}

  async list(): Promise<Page<Operation>> {
    const res = await this.client.get<{ data: { operations: Operation[] } }>(OPERATIONS_PATH)
    const items = res.data?.data?.operations ?? []
    return {
      data: items.map(o => ({ data: o, metadata: {} })),
      loading: false,
      page: { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
      error: null,
    }
  }

  async read(name: string): Promise<Document<Operation> | undefined> {
    try {
      const res = await this.client.get<{ data: Operation }>(
        `${OPERATIONS_PATH}/${encodeURIComponent(name)}`,
      )
      if (!res.data?.data) return undefined
      return { data: res.data.data, metadata: {} }
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND") return undefined
      throw err
    }
  }
}
