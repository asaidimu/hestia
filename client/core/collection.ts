import type { QueryDSL } from "@asaidimu/query";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { HestiaNetworkClient } from "./client";
import { createPagedController, type PageOptions } from "./pager";
import type {
    Document,
    Page,
    PagedData,
    PaginationInfo,
    StoreEvent,
} from "./types";

interface ServerEnvelope<T extends Record<string, any>> {
  data: Document<T>[];
  metadata?: { page?: PaginationInfo };
}

interface SingleEnvelope<T extends Record<string, any>> {
  data: Document<T>;
}

export class HestiaCollection<T extends Record<string, any>> {
  private pagerOptions: PageOptions<T> = {};
  private pager: PagedData<T>;

  constructor(
    private client: HestiaNetworkClient,
    private collectionName: string,
  ) {
    this.pager = createPagedController<T>(
      collectionName,
      new ReactiveDataStore<any>({}),
      this.pagerOptions,
      (query) => this.find(query),
    );
  }

  name() {
    return this.collectionName;
  }

  private get queryPath(): string {
    return `/system/collections/document/${encodeURIComponent(this.collectionName)}/query`;
  }

  private get documentsPath(): string {
    return `/system/collections/document/${encodeURIComponent(this.collectionName)}`;
  }

  private documentPath(id: string): string {
    return `${this.documentsPath}/${encodeURIComponent(id)}`;
  }

  async find(query?: QueryDSL<T>): Promise<Page<T>> {
    const res = await this.client.post<ServerEnvelope<T>>(
      this.queryPath,
      query ?? {},
    );

    const items = res.data?.data ?? [];
    const pageMeta = res.data?.metadata?.page ?? {
      number: 1,
      size: items.length,
      count: items.length,
      total: items.length,
      pages: 1,
    };

    return { data: items, loading: false, page: pageMeta, error: null };
  }

  async read(id: string): Promise<Document<T> | undefined> {
    try {
      const res = await this.client.get<{ data: Document<T> }>(
        this.documentPath(id),
      );
      return res.data?.data;
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND")
        return undefined;
      throw err;
    }
  }

  async create(data: Partial<T>): Promise<Document<T>> {
    const res = await this.client.post<SingleEnvelope<T>>(
      this.documentsPath,
      data,
    );
    return res.data!.data;
  }

  async update(id: string, data: Partial<T>): Promise<Document<T>> {
    const res = await this.client.patch<SingleEnvelope<T>>(
      this.documentPath(id),
      data,
    );
    return res.data!.data;
  }

  async delete(id: string): Promise<void> {
    await this.client.delete(this.documentPath(id));
  }

  async subscribe(
    _scope: string,
    _callback: (event: StoreEvent) => void,
  ): Promise<() => void> {
    throw new Error("Subscription not implemented for dynamic collections");
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not implemented for dynamic collections");
  }

  page(): PagedData<T> {
    return this.pager;
  }
}

export type { ServerEnvelope, SingleEnvelope };
