import { describe, expect, it } from "vitest"
import { makeClient } from "../../tests/helpers"

// E2E against the test server. `markRead`/`markAllRead` are excluded because the
// server writes an `expires_at` field that the `_notifications_` schema rejects
// (ERR_PERSISTENCE_VALIDATION_FAILED) — tracked server-side.

describe("HestiaNotificationStore — E2E", () => {
  const container = makeClient()

  it("lists notifications", async () => {
    const result = await container.notifications.list()
    expect(Array.isArray(result)).toBe(true)
  })

  it("counts unread notifications", async () => {
    const count = await container.notifications.countUnread()
    expect(typeof count).toBe("number")
  })
})