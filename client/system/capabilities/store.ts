import { HestiaNetworkClient } from "../../core/client";
import type { Document } from "../../core/types";

export class HestiaCapabilities {
  constructor(
    private client: HestiaNetworkClient,
  ) {

  }

  async fetch(): Promise<Array<Document<any>> | undefined> {
    try {
      const res = await this.client.get<{ data: Array<Document<any>> }>(
        `/system/core/docs`,
      );
      return res.data?.data;
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF") return undefined;
      throw err;
    }
  }

}
