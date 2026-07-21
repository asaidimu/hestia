import { type Transport } from "../../core/client";
import type { Document, Page, PagedData, StoreEvent } from "../../core/types";
import type { DocumentStore } from "../../core/types";

export class HestiaCapabilities implements DocumentStore<any, Record<string, unknown>, string, Record<string, unknown>, Record<string, unknown>, string, string, Record<string, unknown>> {
  constructor(
    private client: Transport,
  ) {

  }

  async find(_query?: Record<string, unknown>): Promise<Page<any>> {
    const res = await this.client.dispatch<{ data: Array<Document<any>> }>(
      "system:core:docs:list",
    )
    const items = res.data?.data ?? [];
    return {
      data: items,
      loading: false,
      page: { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
    };
  }

  async read(_id: string): Promise<Document<any> | undefined> {
    throw new Error("Read by ID not supported for capabilities; use find()")
  }

  async create(_props: { data: Partial<any> }): Promise<Document<any> | undefined> {
    throw new Error("Capabilities are read-only")
  }

  async update(_props: { data: Partial<any>; options?: string }): Promise<Document<any> | undefined> {
    throw new Error("Capabilities are read-only")
  }

  async delete(_id: string): Promise<void> {
    throw new Error("Capabilities are read-only")
  }

  async list(_options?: Record<string, unknown>): Promise<Page<any>> {
    return this.find()
  }

  async upload(_props: { file: File }): Promise<Document<any> | undefined> {
    throw new Error("Upload not supported for capabilities")
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for capabilities")
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for capabilities")
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<any>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for capabilities")
  }

  page(_options?: Record<string, unknown>): PagedData<any> {
    throw new Error("Pagination not supported for capabilities")
  }
}
