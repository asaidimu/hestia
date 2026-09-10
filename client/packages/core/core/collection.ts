import { ReactiveDataStore } from "@asaidimu/utils-store";
import { type Transport } from "./client";
import { createPagedController, type PageOptions } from "./pager";
import type {
    Document,
    Page,
    PagedData,
    PaginationInfo,
    StoreEvent,
} from "./types";
import type { DocumentStore } from "./types";
import type { QueryDSL, QueryFilter } from "@asaidimu/query";
import { proxiedDocument, proxiedPage, strippedData, type ProxiedDocument } from "./document";

interface ServerEnvelope<T extends Record<string, any>> {
  data: Document<T>[];
  metadata?: { page?: PaginationInfo };
}

interface SingleEnvelope<T extends Record<string, any>> {
  data: Document<T>;
}

export class HestiaCollection<T extends Record<string, any>> implements DocumentStore<T, Record<string, unknown>, string, Record<string, unknown>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pagerOptions: PageOptions<T> = {};
  // @note #qhx9r0 issue : Impotent Page options
  //
  // There is effectively no way to pass in custom pager options
  private pager: PagedData<T>;

  constructor(
    private client: Transport,
    private collectionName: string,
    private defaultLimit: number = 50,
  ) {
    this.pager = createPagedController<T>(
      collectionName,
      new ReactiveDataStore<any>({}),
      this.pagerOptions,
      (query) => this.find(query as any),
    );
  }

  name() {
    return this.collectionName;
  }

  async find(query?: QueryDSL<T>): Promise<Page<ProxiedDocument<T>>> {
    const res = await this.client.dispatch<ServerEnvelope<T>>(
      "system:collections:document:query",
      { arguments: { name: this.collectionName }, payload: query ?? {} },
    );

    const items = res.data?.data ?? [];
    const pageMeta = res.data?.metadata?.page ?? {
      number: 1,
      size: items.length,
      count: items.length,
      total: items.length,
      pages: 1,
    };

    return proxiedPage({ data: items, loading: false, page: pageMeta, error: null });
  }

  async read(id: string): Promise<ProxiedDocument<T> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: Document<T> }>(
        "system:collections:document:get",
        { arguments: { name: this.collectionName, doc_id: id } },
      );
      const doc = res.data?.data;
      return doc ? proxiedDocument(doc) : undefined;
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND")
        return undefined;
      throw err;
    }
  }

  async create(props: { data: Partial<T> }): Promise<ProxiedDocument<T> | undefined> {
    const res = await this.client.dispatch<SingleEnvelope<T>>(
      "system:collections:document:create",
      { arguments: { name: this.collectionName }, payload: strippedData(props.data) },
    );
    return res.data ? proxiedDocument(res.data.data) : undefined;
  }

  async update(props: { data: Partial<T>; id?: string; filter?: QueryFilter<T> }): Promise<ProxiedDocument<T> | undefined> {
    const hasId = !!props.id;
    const hasFilter = !!props.filter;
    if (hasId === hasFilter) {
      throw new Error("update requires exactly one of id or filter");
    }
    // doc_id present → document:update with a raw partial body.
    // doc_id absent → document:update_many with a { set, filter } envelope
    // (its route carries no /{doc_id} segment).
    // Envelope keys are stripped from the written data in both cases —
    // only the server assigns _id_/_metadata_.
    const set = strippedData(props.data);
    if (hasFilter) {
      const res = await this.client.dispatch<SingleEnvelope<T>>(
        "system:collections:document:update_many",
        { arguments: { name: this.collectionName }, payload: { set, filter: props.filter } },
      );
      return res.data ? proxiedDocument(res.data.data) : undefined;
    }
    const res = await this.client.dispatch<SingleEnvelope<T>>(
      "system:collections:document:update",
      { arguments: { name: this.collectionName, doc_id: props.id as string }, payload: set },
    );
    return res.data ? proxiedDocument(res.data.data) : undefined;
  }

  async delete(id: string): Promise<void> {
    await this.client.dispatch("system:collections:document:delete", {
      arguments: { name: this.collectionName, doc_id: id },
    });
  }

  async list(options?: Record<string, unknown>): Promise<Page<ProxiedDocument<T>>> {
    return this.find(
      options ?? { pagination: { type: "offset", offset: 0, limit: this.defaultLimit } },
    );
  }

  async upload(_props: { file: File }): Promise<Document<T> | undefined> {
    throw new Error("Upload not supported for collections");
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

  stream(
    _options: Record<string, unknown>,
    _onStreamChange: () => void,
  ): {
    stream: () => AsyncIterable<Document<T>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for collections");
  }

  page(_options?: Record<string, unknown>): PagedData<T> {
    return this.pager;
  }
}

export type { ServerEnvelope, SingleEnvelope };
