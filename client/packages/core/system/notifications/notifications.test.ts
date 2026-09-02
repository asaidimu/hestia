import { describe, expect, it, vi } from "vitest"
import { makeClient } from "../../tests/helpers"
import { uniqueId } from "../../tests/helpers"
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
    const first = items[0]
    if (!first) return
    await expect(container.notifications.markRead(first._id_)).resolves.toBeUndefined()
  })

  it("marks all notifications as read", async () => {
    await expect(container.notifications.markAllRead()).resolves.toBeUndefined()
  })

  it("creates a notification and lists it back", async () => {
    const subject = uniqueId("e2e-create")
    const created = await container.notifications.create({
      user_id: "auth_disabled",
      subject,
      type: "deploy",
      body: "v1.2.0 is live",
      data: { version: "1.2.0" },
      actions: [{ label: "View changelog", url: "https://example.com/changelog" }],
    })
    expect(created._id_).toBeTruthy()
    expect(created.subject).toBe(subject)
    expect(created.type).toBe("deploy")
    expect(created.read).toBe(false)
    expect(created.actions).toHaveLength(1)
    expect(created.actions?.[0]?.url).toBe("https://example.com/changelog")

    const listed = await container.notifications.list()
    const match = listed.find((n) => n._id_ === created._id_)
    expect(match?.subject).toBe(subject)
  })

  it("defaults type to manual and omits optional fields", async () => {
    const subject = uniqueId("e2e-minimal")
    const created = await container.notifications.create({
      user_id: "auth_disabled",
      subject,
    })
    expect(created.type).toBe("manual")
    expect(created.body).toBeUndefined()
  })

  it("rejects a create without user_id or subject", async () => {
    await expect(
      container.notifications.create({ subject: "no user" } as never),
    ).rejects.toThrow()
    await expect(
      container.notifications.create({ user_id: "auth_disabled" } as never),
    ).rejects.toThrow()
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