import { type Transport } from "../../core/client"
import { HestiaCollection } from "../../core/collection"
import type { Document, Page, PagedData, StoreEvent } from "../../core/types"
import type { DocumentStore } from "../../core/types"
import type { CollectionMeta } from "./types"
import type { SchemaDefinition } from "@asaidimu/utils-schema"

export class HestiaCollections implements DocumentStore<CollectionMeta, Record<string, unknown>, string, Record<string, unknown>, Record<string, unknown>, string, string, Record<string, unknown>> {
  constructor(private client: Transport) {}

  async find(_query?: Record<string, unknown>): Promise<Page<CollectionMeta>> {
    const res = await this.client.dispatch<{
      data: Document<CollectionMeta>[]
    }>("system:collections:collection:list")
    const items = res.data?.data ?? []
    return {
      data: items,
      loading: false,
      page: { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
    }
  }

  async read(name: string): Promise<Document<CollectionMeta> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: Document<CollectionMeta> }>(
        "system:collections:collection:get",
        { arguments: { name } },
      )
      if (!res.data) return undefined
      return res.data.data
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF") return undefined
      throw err
    }
  }

  async create(props: { data: Partial<CollectionMeta> }): Promise<Document<CollectionMeta> | undefined> {
    const schema = (props.data.schema ?? props.data) as unknown as SchemaDefinition
    const res = await this.client.dispatch<{ data: Document<{ schema: any }> }>(
      "system:collections:collection:create",
      { payload: schema },
    )
    return res.data!.data as any as Document<CollectionMeta>
  }

  async update(_props: { data: Partial<CollectionMeta>; options?: string }): Promise<Document<CollectionMeta> | undefined> {
    throw new Error("Collection update not implemented")
  }

  async delete(name: string): Promise<void> {
    await this.client.dispatch("system:collections:collection:delete", {
      arguments: { name },
    })
  }

  async list(_options?: Record<string, unknown>): Promise<Page<CollectionMeta>> {
    return this.find()
  }

  async upload(_props: { file: File }): Promise<Document<CollectionMeta> | undefined> {
    throw new Error("Upload not supported for collections")
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for collections")
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for collections")
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<CollectionMeta>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for collections")
  }

  page(_options?: Record<string, unknown>): PagedData<CollectionMeta> {
    throw new Error("Pagination not supported for collection metadata; use documents(name)")
  }

  documents<T extends Record<string, any>>(collectionName: string): HestiaCollection<T> {
    return new HestiaCollection<T>(this.client, collectionName)
  }
}
