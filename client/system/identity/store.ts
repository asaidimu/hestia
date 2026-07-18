import type { QueryDSL } from "@asaidimu/query";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { HestiaNetworkClient } from "../../core/client";
import { createPagedController } from "../../core/pager";
import type { Document, Page, PagedData, StoreEvent } from "../../core/types";
import type { DocumentStore } from "../../core/types";
import type { UpdateUserRequest, UserData } from "./types";

export class HestiaUsers implements DocumentStore<UserData, QueryDSL<UserData>, string, QueryDSL<UserData>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pagerOptions: any = {};
  private pager: PagedData<UserData>;

  constructor(
    private client: HestiaNetworkClient,
  ) {
    this.pager = createPagedController<UserData>(
      "users",
      new ReactiveDataStore<any>({}),
      this.pagerOptions,
      (query) => this.find(query),
    );
  }

  name() {
    return "users";
  }

  async find(query?: QueryDSL<UserData>): Promise<Page<UserData>> {
    const res = await this.client.post<{
      data:  Document<UserData>[];
      metadata?: { page?: any };
    }>("/system/users/user/query", query ?? {});
    const data = res.data?.data ?? [];
    const pageMeta = res.data?.metadata?.page ?? {
      number: 1,
      size: data.length,
      count: data.length,
      total: data.length,
      pages: 1,
    };
    return { data, loading: false, page: pageMeta };
  }

  async list(options?: QueryDSL<UserData>): Promise<Page<UserData>> {
    return this.find(
      options ?? { pagination: { type: "offset", offset: 0, limit: 50 } },
    );
  }

  async read(id: string): Promise<Document<UserData> | undefined> {
    try {
      const res = await this.client.get<{ data: Document<UserData> }>(
        `/system/users/user/${encodeURIComponent(id)}`,
      );
      return res.data?.data;
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF") return undefined;
      throw err;
    }
  }

  async update(props: { data: Partial<UserData>; options?: string }): Promise<Document<UserData> | undefined> {
    const id = props.options!;
    const res = await this.client.patch<{ data: Document<UserData> }>(
      `/system/users/user/${encodeURIComponent(id)}`,
      props.data as any,
    );
    return res.data!.data;
  }

  async delete(id: string): Promise<void> {
    await this.client.delete(`/system/users/user/${encodeURIComponent(id)}`);
  }

  async create(props: { data: Partial<UserData> }): Promise<Document<UserData> | undefined> {
    throw new Error("User creation requires email/password/name, use register endpoint");
  }

  async upload(_props: { file: File }): Promise<Document<UserData> | undefined> {
    throw new Error("Upload not supported for users");
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for users");
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for users");
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<UserData>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for users");
  }

  page(_options?: Record<string, unknown>): PagedData<UserData> {
    return this.pager;
  }

  async changePassword(
    userId: string,
    current: string,
    newPassword: string,
  ): Promise<void> {
    await this.client.patch(`/system/users/password/${encodeURIComponent(userId)}`, {
      current,
      new: newPassword,
    });
  }
}