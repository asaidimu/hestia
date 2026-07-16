import type { QueryDSL } from "@asaidimu/query";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { HestiaNetworkClient } from "../core/client";
import { createPagedController } from "../core/pager";
import type { Page, PagedData } from "../core/types";
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

/**
 * Namespace-scoped blob facade shaped like HestiaCollection.
 * Wraps blob CRUD behind a DocumentStore-compatible interface
 * so you can use DataTable + PagedData.
 */
export class BlobNamespace {
  private pagerOptions = {};
  private pager: PagedData<BlobMeta>;
  private prefixFilter = "";

  constructor(
    private client: HestiaNetworkClient,
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

  private basePath() {
    return `/system/blobs/blob/${encodeURIComponent(this.ns)}`;
  }

  async find(query?: QueryDSL<BlobMeta>): Promise<Page<BlobMeta>> {
    const prefix = this.prefixFilter || (query as any)?.prefix || "";
    const limit =
      (query as any)?.limit ?? query?.pagination?.limit ?? 0;

    const req: ListBlobsRequest = {};
    if (prefix) req.prefix = prefix;
    if (limit) req.limit = limit;

    const res = await this.client.post<{
      data: { blobs: BlobMeta[] };
    }>(`${this.basePath()}/query`, req);

    const items = res.data?.data?.blobs ?? [];
    return { data: items.map(asDoc), loading: false, page: pageMeta(items), error: undefined };
  }

  async head(key: string): Promise<BlobDocument | undefined> {
    try {
      const res = await this.client.get<{ data: BlobMeta }>(
        `${this.basePath()}/${encodeURIComponent(key)}`,
      );
      if (!res.data?.data) return undefined;
      return asDoc(res.data.data);
    } catch (err: any) {
      if (
        err?.code === "SYNC-001-NF" ||
        (err?.code === "INTERNAL_ERROR" &&
          typeof err?.message === "string" &&
          err.message.includes("not found"))
      )
        return undefined;
      throw err;
    }
  }

  async upload(
    key: string,
    data: Blob,
    contentType?: string,
  ): Promise<BlobDocument> {
    const headers: Record<string, string> = {};
    const ct = contentType || data.type;
    if (ct) headers["Content-Type"] = ct;

    const res = await this.client.post<{ data: BlobMeta }>(
      `${this.basePath()}/${encodeURIComponent(key)}`,
      data,
      { headers, bodyType: "blob" },
    );
    return asDoc(res.data!.data);
  }

  async download(
    key: string,
  ): Promise<{ data: Blob; contentType: string }> {
    const res = await this.client.get<Blob>(
      `${this.basePath()}/${encodeURIComponent(key)}`,
      { responseType: "blob" },
    );
    const blob = res.data!;
    return { data: blob, contentType: blob.type };
  }

  async updateMetadata(key: string, custom: Record<string, any>): Promise<BlobMeta> {
    const res = await this.client.patch<{ data: BlobMeta }>(
      `${this.basePath()}/${encodeURIComponent(key)}`,
      { custom },
    );
    return res.data!.data;
  }

  async delete(key: string): Promise<void> {
    await this.client.delete(
      `${this.basePath()}/${encodeURIComponent(key)}`,
    );
  }

  async list(): Promise<BlobMeta[]> {
    const res = await this.client.post<{
      data: { blobs: BlobMeta[] };
    }>(`${this.basePath()}/query`, {});
    return res.data?.data?.blobs ?? [];
  }

  page(): PagedData<BlobMeta> {
    return this.pager;
  }
}

/**
 * Top-level blob client. Entry point for all blob operations.
 *
 * Usage:
 *   client.blobs.listNamespaces()
 *   const ns = client.blobs.namespace("my-bucket")
 *   ns.find({ prefix: "images/" })
 *   ns.upload("logo.png", file)
 *   const { data } = await ns.download("logo.png")
 */
export class HestiaBlobClient {
  constructor(private client: HestiaNetworkClient) {}

  private nsBase = "/system/blobs";

  // ── Namespace operations ──────────────────────────────────────────────

  async namespaces(): Promise<NamespaceInfo[]> {
    const res = await this.client.post<{
      data: { namespaces: NamespaceInfo[] };
    }>(`${this.nsBase}/namespace/query`);
    return res.data?.data?.namespaces ?? [];
  }

  async createNamespace(data: CreateNamespaceRequest): Promise<NamespaceInfo> {
    const res = await this.client.post<{ data: NamespaceInfo }>(
      `${this.nsBase}/namespace`,
      data,
    );
    return res.data!.data;
  }

  async deleteNamespace(ns: string): Promise<void> {
    await this.client.delete(
      `${this.nsBase}/namespace/${encodeURIComponent(ns)}`,
    );
  }


  blob(namespace: string, key:string) {
      return `${this.client.base()}${this.nsBase}/blob/${encodeURIComponent(namespace)}/${encodeURIComponent(key)}`
  }
  // ── Namespace-scoped facade ───────────────────────────────────────────

  namespace(ns: string): BlobNamespace {
    return new BlobNamespace(this.client, ns);
  }
}
