import type { Document } from "../../core/types"
import { type Transport } from "../../core/client"
import type { Schedule, CreateSchedulePayload, UpdateSchedulePayload } from "./types"

/** CRUD store for cron-triggered schedules. */
export class HestiaScheduleStore {
  constructor(private client: Transport) {}

  /** Create a new schedule and register the cron job. Returns the new schedule ID. */
  async create(payload: CreateSchedulePayload): Promise<string> {
    const res = await this.client.dispatch<{ data: { id: string } }>(
      "system:schedules:schedule:create",
      { payload },
    )
    return res.data?.data?.id ?? ""
  }

  /** List all schedules visible to the current tenant. */
  async list(): Promise<Document<Schedule>[]> {
    const res = await this.client.dispatch<{ data: Document<Schedule>[] }>(
      "system:schedules:schedule:list",
    )
    return res.data?.data ?? []
  }

  /** Get a single schedule by ID. Returns null if not found. */
  async get(id: string): Promise<Document<Schedule> | null> {
    const res = await this.client.dispatch<{ data: Document<Schedule> }>(
      "system:schedules:schedule:get",
      { arguments: { id } },
    )
    return res.data?.data ?? null
  }

  /** Update fields on a schedule. The cron job is re-registered with the new values. */
  async update(id: string, payload: UpdateSchedulePayload): Promise<void> {
    await this.client.dispatch("system:schedules:schedule:update", {
      arguments: { id },
      payload,
    })
  }

  /** Delete a schedule and unregister its cron job. */
  async delete(id: string): Promise<void> {
    await this.client.dispatch("system:schedules:schedule:delete", {
      arguments: { id },
    })
  }
}
