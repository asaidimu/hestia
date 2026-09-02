import type { Document } from "../../core/types"
import { type Transport } from "../../core/client"
import type { SettingDocument } from "./types"

export class HestiaSettingStore {
  constructor(private client: Transport) {}

  async list(): Promise<Document<SettingDocument>[]> {
    const res = await this.client.dispatch<{ data: Document<SettingDocument>[] }>(
      "system:settings:list",
    )
    return res.data?.data ?? []
  }

  async get(key: string): Promise<Document<SettingDocument> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: Document<SettingDocument> }>(
        "system:settings:get",
        { arguments: { key } },
      )
      return res.data?.data
    } catch {
      return undefined
    }
  }

  async set(key: string, value: Record<string, unknown>): Promise<void> {
    await this.client.dispatch("system:settings:set", {
      arguments: { key },
      payload: { value },
    })
  }

  async delete(key: string): Promise<void> {
    await this.client.dispatch("system:settings:delete", {
      arguments: { key },
    })
  }
}
