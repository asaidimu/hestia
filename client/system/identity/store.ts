import type { QueryDSL } from "@asaidimu/query";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { type Transport } from "../../core/client";
import { createPagedController } from "../../core/pager";
import type { Document, Page, PagedData, StoreEvent } from "../../core/types";
import type { DocumentStore } from "../../core/types";
import type { UpdateUserRequest, UserData } from "./types";

export class HestiaUsers implements DocumentStore<UserData, QueryDSL<UserData>, string, QueryDSL<UserData>, Record<string, unknown>, string, string, Record<string, unknown>> {
  private pagerOptions: any = {};
  private pager: PagedData<UserData>;

  constructor(
    private client: Transport,
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
    const res = await this.client.dispatch<{
      data: Document<UserData>[];
      metadata?: { page?: any };
    }>("system:users:user:query", { payload: query ?? {} });
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
      const res = await this.client.dispatch<{ data: Document<UserData> }>(
        "system:users:user:get",
        { arguments: { user_id: id } },
      );
      return res.data?.data;
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF") return undefined;
      throw err;
    }
  }

  async update(props: { data: Partial<UserData>; options?: string }): Promise<Document<UserData> | undefined> {
    const id = props.options!;
    const res = await this.client.dispatch<{ data: Document<UserData> }>(
      "system:users:user:update",
      { arguments: { user_id: id }, payload: props.data as any },
    );
    return res.data!.data;
  }

  async delete(id: string): Promise<void> {
    await this.client.dispatch("system:users:user:delete", {
      arguments: { user_id: id },
    });
  }

  async create(_props: { data: Partial<UserData> }): Promise<Document<UserData> | undefined> {
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
    await this.client.dispatch("system:users:password:change", {
      arguments: { user_id: userId },
      payload: { current, new: newPassword },
    });
  }
}
