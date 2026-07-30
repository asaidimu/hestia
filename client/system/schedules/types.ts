/** A schedule document stored in the system. */
export interface Schedule {
  _id_: string
  /** Owner of this schedule. */
  user_id: string
  /** System message name to dispatch when the cron fires. */
  message: string
  /** Input payload passed to the message handler. String values may contain Go templates (`{{ .schedule._id }}`, `{{ .now }}`). */
  input?: Record<string, unknown>
  /** Cron expression (e.g. `@every 5m`, `0 * * * *`). */
  cron: string
  /** When true the schedule is paused and will not fire. */
  disabled?: boolean
  tenant_id?: string
  created_at: number
  _metadata_: Record<string, unknown>
}

/** Payload for `schedule:create`. */
export interface CreateSchedulePayload {
  /** Override the owner (defaults to the authenticated user). */
  user_id?: string
  /** System message name to dispatch on each tick. */
  message: string
  /** Input payload. String values are resolved as Go templates against `{ .schedule, .now }`. */
  input?: Record<string, unknown>
  /** Cron expression. */
  cron: string
  /** Start the schedule in a disabled state. */
  disabled?: boolean
}

/** Payload for `schedule:update`. All fields optional — only provided fields are patched. */
export interface UpdateSchedulePayload {
  message?: string
  input?: Record<string, unknown>
  cron?: string
  disabled?: boolean
}
