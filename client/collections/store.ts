import { HestiaNetworkClient } from "../core/client"
import { HestiaCollection } from "../core/collection"
import type { Document } from "../core/types"
import type { CollectionMeta } from "./types"

export class HestiaCollections {
  constructor(private client: HestiaNetworkClient) {}

  async list(): Promise<{ collections: CollectionMeta[]; total: number }> {
    const res = await this.client.get<{
      data: { name: string; schema: any; created: string; updated: string }[]
    }>("/system/collections/collection")
    const items = res.data?.data ?? []
    return {
      collections: items.map((i) => ({
        name: i.name,
        schema: i.schema,
        created: i.created,
        updated: i.updated,
      })),
      total: items.length,
    }
  }

  async read(name: string): Promise<CollectionMeta | undefined> {
    try {
      const res = await this.client.get<{ data: { name: string; schema: any; created: string; updated: string } }>(
        `/system/collections/collection/${encodeURIComponent(name)}`,
      )
      if (!res.data) return undefined
      return res.data.data
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF") return undefined
      throw err
    }
  }

  async create(schema: any): Promise<Document<{ schema: any }>> {
    const res = await this.client.post<{ data: Document<{ schema: any }> }>("/system/collections/collection", schema)
    return res.data!.data
  }

  async update(name: string, schema: any): Promise<Document<{ schema: any }>> {
      throw new Error("Method not implemented")
    // const res = await this.client.patch<{ data: Document<{ schema: any }> }>(
    //   `/api/admin/collections/${encodeURIComponent(name)}`,
    //   schema,
    // )
    // return res.data!.data
  }

  async delete(name: string): Promise<void> {
    await this.client.delete(`/system/collections/collection/${encodeURIComponent(name)}`)
  }

  documents<T extends Record<string, any>>(collectionName: string): HestiaCollection<T> {
    return new HestiaCollection<T>(this.client,collectionName)
  }
}
