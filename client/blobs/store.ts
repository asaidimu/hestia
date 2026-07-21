import type { QueryDSL } from "@asaidimu/query";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { type Transport } from "../core/client";
import { createPagedController } from "../core/pager";
import type { Document, Page, PagedData, StoreEvent } from "../core/types";
import type { DocumentStore } from "../core/types";
import type {
    BlobDocument,
    BlobMeta,
    CreateNamespaceRequest,
    ListBlobsRequest,
    NamespaceInfo,
} from "./types";

function asDoc(b: BlobMeta): BlobDocument {
  return {
    _id_: b.key,
    _metadata_: {
      checksum: "",
      created: b.created_at,
      updated: b.updated_at ?? b.created_at,
      version: 1,
    },
    ...b,
  };
}

function pageMeta<T extends Record<string,any>>(items: T[]): Page<T>["page"] {
  return {
    number: 1,
    size: items.length,
    count: items.length,
    total: items.length,
    pages: 1,
  };
}

export class BlobNamespace implements DocumentStore<BlobMeta, QueryDSL<BlobMeta>, string, QueryDSL<BlobMeta>, Record<string, unknown>, string, Record<string, any>, Record<string, unknown>, { key: string; contentType?: string }, Record<string, unknown>> {
  private pagerOptions = {};
  private pager: PagedData<BlobMeta>;
  private prefixFilter = "";

  constructor(
    private client: Transport,
    private ns: string,
  ) {
    this.pager = createPagedController<BlobMeta>(
      `blobs_${ns}`,
      new ReactiveDataStore<any>({}),
      this.pagerOptions,
      (query) => this.find(query),
    );
  }

  name() {
    return this.ns;
  }

  setPrefix(prefix: string) {
    this.prefixFilter = prefix;
  }

  async find(query?: QueryDSL<BlobMeta>): Promise<Page<BlobMeta>> {
    const prefix = this.prefixFilter || (query as any)?.prefix || "";
    const limit =
      (query as any)?.limit ?? query?.pagination?.limit ?? 0;

    const req: ListBlobsRequest = {};
    if (prefix) req.prefix = prefix;
    if (limit) req.limit = limit;

    const res = await this.client.dispatch<{
      data: { blobs: BlobMeta[] };
    }>("system:blobs:blob:list", {
      arguments: { ns: this.ns },
      payload: req,
    });

    const items = res.data?.data?.blobs ?? [];
    return { data: items.map(asDoc), loading: false, page: pageMeta(items), error: undefined };
  }

  async read(key: string): Promise<Document<BlobMeta> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: BlobMeta }>(
        "system:blobs:blob:head",
        { arguments: { ns: this.ns, key } },
      );
      if (!res.data?.data) return undefined;
      return asDoc(res.data.data);
    } catch (err: any) {
      if (err?.code === "NOT_FOUND") return undefined;
      throw err;
    }
  }

  async create(_props: { data: Partial<BlobMeta> }): Promise<Document<BlobMeta> | undefined> {
    throw new Error("Use upload() to create blobs");
  }

  async update(props: { data: Partial<BlobMeta>; options?: Record<string, any> }): Promise<Document<BlobMeta> | undefined> {
    const key = props.options?.key as string;
    if (!key) throw new Error("options.key is required for blob update");
    const res = await this.client.dispatch<{ data: BlobMeta }>(
      "system:blobs:blob:update",
      { arguments: { ns: this.ns, key }, payload: { custom: props.data } },
    );
    return asDoc(res.data!.data);
  }

  async delete(key: string): Promise<void> {
    await this.client.dispatch("system:blobs:blob:delete", {
      arguments: { ns: this.ns, key },
    });
  }

  async list(options?: QueryDSL<BlobMeta>): Promise<Page<BlobMeta>> {
    return this.find(options ?? {});
  }

  async upload(props: { file: File; options?: { key?: string; contentType?: string } }): Promise<Document<BlobMeta> | undefined> {
    const key = (props.options as any)?.key as string;
    if (!key) throw new Error("options.key is required for blob upload");
    const headers: Record<string, string> = {};
    const ct = props.options?.contentType || props.file.type;
    if (ct) headers["Content-Type"] = ct;

    const res = await this.client.dispatch<{ data: BlobMeta }>(
      "system:blobs:blob:upload",
      { arguments: { ns: this.ns, key }, payload: props.file, headers, bodyType: "blob" },
    );
    return asDoc(res.data!.data);
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for blobs");
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for blobs");
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<BlobMeta>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for blobs");
  }

  page(_options?: Record<string, unknown>): PagedData<BlobMeta> {
    return this.pager;
  }

  async download(key: string): Promise<{ data: Blob; contentType: string }> {
    const res = await this.client.dispatch<Blob>(
      "system:blobs:blob:download",
      { arguments: { ns: this.ns, key }, responseType: "blob" },
    );
    const blob = res.data!;
    return { data: blob, contentType: blob.type };
  }
}

export class HestiaBlobClient {
  private apiPrefix: string;

  constructor(private client: Transport, apiPrefix: string = "/api") {
    this.apiPrefix = apiPrefix;
  }

  async namespaces(): Promise<NamespaceInfo[]> {
    const res = await this.client.dispatch<{
      data: { namespaces: NamespaceInfo[] };
    }>("system:blobs:namespace:list");
    return res.data?.data?.namespaces ?? [];
  }

  async createNamespace(data: CreateNamespaceRequest): Promise<NamespaceInfo> {
    const res = await this.client.dispatch<{ data: NamespaceInfo }>(
      "system:blobs:namespace:create",
      { payload: data },
    );
    return res.data!.data;
  }

  async deleteNamespace(ns: string): Promise<void> {
    await this.client.dispatch("system:blobs:namespace:delete", {
      arguments: { ns },
    });
  }

  blob(namespace: string, key:string) {
      return `${this.client.base()}${this.apiPrefix}/system/blobs/blob/${encodeURIComponent(namespace)}/${encodeURIComponent(key)}`
  }

  namespace(ns: string): BlobNamespace {
    return new BlobNamespace(this.client, ns);
  }
}
