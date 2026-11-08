import { describe, expect, it } from "vitest"
import { HestiaClient } from "../../container"
import type { Document } from "../../core/types"
import type { Notification } from "./types"

const BASE_URL = "http://localhost:8070"

function deferred<T = void>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((r) => {
    resolve = r
  })
  return { promise, resolve }
}

async function waitUntil(predicate: () => boolean, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (predicate()) return
    await new Promise((r) => setTimeout(r, 25))
  }
  if (!predicate()) throw new Error("timed out waiting for condition")
}

describe("notifications SSE — end to end", () => {
  const container = new HestiaClient({ baseUrl: BASE_URL })

  it("delivers a created notification to a live stream", async () => {
    await container.auth.login("admin@test.local", "password123")

    const opened = deferred()
    const received: Document<Notification>[] = []
    let streamError: Error | null = null

    const controller = container.notifications.stream({
      onOpen: () => opened.resolve(),
      onMessage: (notification) => received.push(notification),
      onError: (err) => {
        streamError = err
      },
    })

    try {
      await opened.promise

      // The test server forces the session user id to "auth_disabled", so the
      // notification must target that user to match the stream's filter.
      const subject = `sse-probe-${Date.now()}`
      await container.notifications.create({
        user_id: "auth_disabled",
        subject,
        body: "stream probe",
        type: "manual",
      })

      await waitUntil(() => received.some((n) => n.subject === subject), 5000)

      expect(received.some((n) => n.subject === subject)).toBe(true)
      expect(streamError).toBeNull()
    } finally {
      controller.abort()
    }
  }, 15000)
})
