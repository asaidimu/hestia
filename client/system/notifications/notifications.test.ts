import { describe, expect, it, vi } from "vitest"
import { makeClient } from "../../tests/helpers"
import { HestiaNotificationStore } from "./store"
import type { NotificationAction } from "./types"

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

  it("marks the first notification as read", async () => {
    const items = await container.notifications.list()
    if (items.length === 0) return
    await expect(container.notifications.markRead(items[0]._id_)).resolves.toBeUndefined()
  })

  it("marks all notifications as read", async () => {
    await expect(container.notifications.markAllRead()).resolves.toBeUndefined()
  })
})

describe("HestiaNotificationStore — dispatchAction", () => {
  it("dispatches the action message with its arguments", async () => {
    const dispatch = vi.fn().mockResolvedValue({ data: {} })
    const store = new HestiaNotificationStore({ dispatch } as any)
    const action: NotificationAction = {
      label: "Apply update",
      message: "system:updates:update:apply",
      arguments: { foo: "bar" },
    }
    await store.dispatchAction(action)
    expect(dispatch).toHaveBeenCalledWith("system:updates:update:apply", {
      arguments: { foo: "bar" },
    })
  })

  it("dispatches with empty arguments when none are attached", async () => {
    const dispatch = vi.fn().mockResolvedValue({ data: {} })
    const store = new HestiaNotificationStore({ dispatch } as any)
    await store.dispatchAction({ label: "Apply update", message: "system:updates:update:apply" })
    expect(dispatch).toHaveBeenCalledWith("system:updates:update:apply", {
      arguments: {},
    })
  })

  it("rejects URL-only actions (no message to dispatch)", async () => {
    const store = new HestiaNotificationStore({ dispatch: vi.fn() } as any)
    await expect(store.dispatchAction({ label: "Open changelog", url: "https://example.com" })).rejects.toThrow(
      "no message",
    )
  })
})