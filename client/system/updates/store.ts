import { type Transport } from "../../core/client";
import type {
  UpdateStatus,
  UpdateChangelog,
  UpdateCheckResult,
  UpdateAvailability,
  UpdateStageResult,
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

  /**
   * Read-only availability check: reports whether a newer version exists
   * without downloading or staging anything. Safe to poll frequently.
   */
  async checkAvailability(): Promise<UpdateAvailability> {
    const res = await this.client.dispatch<{ data: UpdateAvailability }>(
      "system:updates:check:get",
    );
    return res.data.data;
  }

  /**
   * Download and stage the newest release without applying it. Notifies
   * administrators when a release was newly staged.
   */
  async stage(): Promise<UpdateStageResult> {
    const res = await this.client.dispatch<{ data: UpdateStageResult }>(
      "system:updates:stage:create",
    );
    return res.data.data;
  }

  /**
   * Legacy check-then-stage: checks for an update, stages it when found, and
   * notifies admins. Prefer {@link checkAvailability} + {@link stage} so each
   * effect stays independently callable.
   */
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
