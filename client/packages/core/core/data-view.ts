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
import type { DataViewMeta } from "../system/collections/types";
import { proxiedDocument, proxiedPage, type ProxiedDocument } from "./document";

interface ServerEnvelope<T extends Record<string, any>> {
  data: Document<T>[];
  metadata?: { page?: PaginationInfo };
}

interface DescriptorEnvelope {
  data: Document<DataViewMeta>;
}

export interface DataViewDefinition<T extends Record<string, any>> {
  name: string;
  /** Stored query. `target.name` pins the underlying collection (required by the server). */
  query: QueryDSL<T> & { target: { name: string } };
  materialized?: boolean;
}

/**
 * Read-only view over a stored server query — the view counterpart to
 * HestiaCollection. Reads flow through the regular document:query /
 * document:get messages addressed at the view name; document writes are
 * rejected server-side (ERR_PERSISTENCE_READ_ONLY) and throw client-side.
 * Materialized snapshots are rebuilt via refresh().
 */
export class HestiaDataView<T extends Record<string, any>> implements DocumentStore<T, Record<string, unknown>, string, Record<string, unknown>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pagerOptions: PageOptions<T> = {};
  private pager: PagedData<T>;

  constructor(
    private client: Transport,
    private viewName: string,
    private defaultLimit: number = 50,
  ) {
    this.pager = createPagedController<T>(
      viewName,
      new ReactiveDataStore<any>({}),
      this.pagerOptions,
      (query) => this.find(query as any),
    );
  }

  /** Registers a view on the server and returns a handle to it. */
  static async create<T extends Record<string, any>>(
    client: Transport,
    definition: DataViewDefinition<T>,
  ): Promise<HestiaDataView<T>> {
    await client.dispatch<DescriptorEnvelope>(
      "system:collections:view:create",
      {
        arguments: { name: definition.name },
        payload: { query: definition.query, materialized: definition.materialized ?? false },
      },
    );
    return new HestiaDataView<T>(client, definition.name);
  }

  name() {
    return this.viewName;
  }

  async find(query?: QueryDSL<T>): Promise<Page<ProxiedDocument<T>>> {
    const res = await this.client.dispatch<ServerEnvelope<T>>(
      "system:collections:document:query",
      { arguments: { name: this.viewName }, payload: query ?? {} },
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
        { arguments: { name: this.viewName, doc_id: id } },
      );
      const doc = res.data?.data;
      return doc ? proxiedDocument(doc) : undefined;
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND")
        return undefined;
      throw err;
    }
  }

  /** Rebuilds a materialized view's snapshot. Fails on virtual views. */
  async refresh(): Promise<ProxiedDocument<DataViewMeta> | undefined> {
    const res = await this.client.dispatch<DescriptorEnvelope>(
      "system:collections:view:refresh",
      { arguments: { name: this.viewName } },
    );
    const doc = res.data?.data;
    return doc ? proxiedDocument(doc) : undefined;
  }

  async create(_props: { data: Partial<T> }): Promise<ProxiedDocument<T> | undefined> {
    throw new Error("Create not supported for views: views are read-only");
  }

  async update(_props: { data: Partial<T>; id?: string; filter?: QueryFilter<T> }): Promise<ProxiedDocument<T> | undefined> {
    throw new Error("Update not supported for views: views are read-only");
  }

  async delete(_id: string): Promise<void> {
    throw new Error("Delete not supported for views: views are read-only");
  }

  async list(options?: Record<string, unknown>): Promise<Page<ProxiedDocument<T>>> {
    return this.find(
      options ?? { pagination: { type: "offset", offset: 0, limit: this.defaultLimit } },
    );
  }

  async upload(_props: { file: File }): Promise<ProxiedDocument<T> | undefined> {
    throw new Error("Upload not supported for views");
  }

  async subscribe(
    _scope: string,
    _callback: (event: StoreEvent) => void,
  ): Promise<() => void> {
    throw new Error("Subscription not implemented for views");
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not implemented for views");
  }

  stream(
    _options: Record<string, unknown>,
    _onStreamChange: () => void,
  ): {
    stream: () => AsyncIterable<Document<T>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for views");
  }

  page(_options?: Record<string, unknown>): PagedData<T> {
    return this.pager;
  }
}
