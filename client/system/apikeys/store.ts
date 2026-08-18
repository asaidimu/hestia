import type { QueryDSL } from "@asaidimu/query"
import { type Transport } from "../../core/client"
import { ReactiveDataStore } from "@asaidimu/utils-store"
import { createPagedController } from "../../core/pager"
import type { Document, Page, PagedData, StoreEvent } from "../../core/types"
import type { DocumentStore } from "../../core/types"
import type { APIKey, APIKeyWithSecret, CreateKeyRequest, UpdateKeyRequest } from "./types"

export class HestiaKeyStore implements DocumentStore<APIKey, QueryDSL<APIKey>, string, QueryDSL<APIKey>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pager: PagedData<APIKey>

  constructor(private client: Transport) {
    this.pager = createPagedController<APIKey>(
      "_api_key_",
      new ReactiveDataStore<any>({}),
      {},
      (query) => this.find(query),
    )
  }

  async find(_query?: QueryDSL<APIKey>): Promise<Page<APIKey>> {
    const res = await this.client.dispatch<{
      data: Document<APIKey>[];
      metadata?: { page?: any };
    }>("system:apikeys:key:list")
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

  async list(options?: QueryDSL<APIKey>): Promise<Page<APIKey>> {
    return options ? this.find(options) : this.find()
  }

  async read(id: string): Promise<Document<APIKey> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: Document<APIKey> }>(
        "system:apikeys:key:get",
        { arguments: { key_id: id } },
      )
      return res.data?.data
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF") return undefined
      if (err?.code === "INTERNAL_ERROR" && typeof err?.message === "string" && err.message.includes("not found")) return undefined
      throw err
    }
  }

  async create(props: { data: Partial<APIKey> }): Promise<Document<APIKey> | undefined> {
    const res = await this.client.dispatch<{ data: Document<APIKeyWithSecret> }>(
      "system:apikeys:key:create",
      { payload: props.data as CreateKeyRequest },
    )
    return res.data!.data
  }

  async update(props: { data: Partial<APIKey>; options?: string }): Promise<Document<APIKey> | undefined> {
    const id = props.options!
    const res = await this.client.dispatch<{ data: Document<APIKey> }>(
      "system:apikeys:key:update",
      { arguments: { key_id: id }, payload: props.data as UpdateKeyRequest },
    )
    return res.data!.data
  }

  async delete(id: string): Promise<void> {
    await this.client.dispatch("system:apikeys:key:delete", {
      arguments: { key_id: id },
    })
  }

  async upload(_props: { file: File }): Promise<Document<APIKey> | undefined> {
    throw new Error("Upload not supported for API keys")
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for API keys")
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for API keys")
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<APIKey>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for API keys")
  }

  page(_options?: Record<string, unknown>): PagedData<APIKey> {
    return this.pager
  }

  async rotate(id: string): Promise<Document<APIKeyWithSecret>> {
    const res = await this.client.dispatch<{ data: Document<APIKeyWithSecret> }>(
      "system:apikeys:key:rotate",
      { arguments: { key_id: id } },
    )
    return res.data!.data
  }
}
