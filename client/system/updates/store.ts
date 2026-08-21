import { type Transport } from "../../core/client";
import type {
  UpdateStatus,
  UpdateChangelog,
  UpdateCheckResult,
  UpdateApplyResult,
} from "./types";

export class HestiaUpdates {
  constructor(private client: Transport) {}

  async status(): Promise<UpdateStatus> {
    const res = await this.client.dispatch<{ data: UpdateStatus }>(
      "system:updates:status:get",
    );
    return res.data.data;
  }

  async changelog(): Promise<UpdateChangelog> {
    const res = await this.client.dispatch<{ data: UpdateChangelog }>(
      "system:updates:changelog:get",
    );
    return res.data.data;
  }

  async check(): Promise<UpdateCheckResult> {
    const res = await this.client.dispatch<{ data: UpdateCheckResult }>(
      "system:updates:check:create",
    );
    return res.data.data;
  }

  async apply(): Promise<UpdateApplyResult> {
    const res = await this.client.dispatch<{ data: UpdateApplyResult }>(
      "system:updates:update:apply",
    );
    return res.data.data;
  }
}
