import { describe, expect, it, afterAll } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"

describe("HestiaScheduleStore — E2E", () => {
  const container = makeClient()
  let scheduleId: string

  afterAll(async () => {
    await container.schedules.delete(scheduleId).catch(() => {})
  })

  it("creates a schedule and returns its id", async () => {
    scheduleId = await container.schedules.create({
      message: "system:core:heartbeat",
      cron: "@every 10m",
      input: {},
    })
    expect(scheduleId).toBeTruthy()
  })

  it("lists schedules", async () => {
    const items = await container.schedules.list()
    expect(items.some((s) => s._id_ === scheduleId)).toBe(true)
  })

  it("gets a schedule by id", async () => {
    const doc = await container.schedules.get(scheduleId)
    expect(doc).not.toBeNull()
    expect(doc!._id_).toBe(scheduleId)
  })

  it("updates a schedule", async () => {
    await container.schedules.update(scheduleId, { cron: "@every 20m" })
    const doc = await container.schedules.get(scheduleId)
    expect(doc!.cron).toBe("@every 20m")
  })

  it("deletes a schedule", async () => {
    await container.schedules.delete(scheduleId)
  })
})