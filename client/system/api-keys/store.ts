import type { QueryDSL } from "@asaidimu/query"
import { HestiaNetworkClient } from "../../core/client"
import type { Document, Page, PagedData } from "../../core/types"
import type { APIKey, APIKeyWithSecret, CreateKeyRequest, UpdateKeyRequest } from "./types"

export class HestiaKeyStore {
  constructor(private client: HestiaNetworkClient) {}

  private basePath = "/system/apikeys/key"

  async find(_query?: QueryDSL<APIKey>): Promise<Page<APIKey>> {
    const res = await this.client.get<{
      data: Document<APIKey>[];
      metadata?: { page?: any };
    }>(this.basePath)
    const items = res.data?.data ?? []
    const pageMeta = res.data?.metadata?.page ?? {
      number: 1,
      size: items.length,
      count: items.length,
      total: items.length,
      pages: 1,
    }
    return { data: items, loading: false, page: pageMeta }
  }

  async list(): Promise<Page<APIKey>> {
    return this.find()
  }

  async read(id: string): Promise<Document<APIKey> | undefined> {
    try {
      const res = await this.client.get<{ data: Document<APIKey> }>(
        `${this.basePath}/${encodeURIComponent(id)}`,
      )
      return res.data?.data
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF") return undefined
      if (err?.code === "INTERNAL_ERROR" && typeof err?.message === "string" && err.message.includes("not found")) return undefined
      throw err
    }
  }

  async create(data: CreateKeyRequest): Promise<Document<APIKeyWithSecret>> {
    const res = await this.client.post<{ data: Document<APIKeyWithSecret> }>(
      this.basePath,
      data,
    )
    return res.data!.data
  }

  async update(id: string, data: UpdateKeyRequest): Promise<Document<APIKey>> {
    const res = await this.client.patch<{ data: Document<APIKey> }>(
      `${this.basePath}/${encodeURIComponent(id)}`,
      data as any,
    )
    return res.data!.data
  }

  async delete(id: string): Promise<void> {
    await this.client.delete(`${this.basePath}/${encodeURIComponent(id)}`)
  }

  async rotate(id: string): Promise<Document<APIKeyWithSecret>> {
    const res = await this.client.post<{ data: Document<APIKeyWithSecret> }>(
      `${this.basePath}/${encodeURIComponent(id)}`,
    )
    return res.data!.data
  }
}
