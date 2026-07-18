import { describe, expect, it } from "vitest"
import { HestiaClient } from "../container"

const BASE_URL = "http://localhost:8070"

describe("SSE stream — audit log", () => {
  const container = new HestiaClient({ baseUrl: BASE_URL })

  it.skip("streams audit log entries after login", async () => {
    await container.auth.login("admin@test.local", "password123")

    const received: string[] = []
    let streamErr: Error | null = null
    const ac = new AbortController()

    const streamPromise = container.client.openStream(
      "/system/audit/log/stream",
      {
        onMessage: (data) => { received.push(data) },
        onError: (err) => { streamErr = err },
        onClose: () => {},
      },
      { signal: ac.signal },
    )

    await new Promise((r) => setTimeout(r, 500))

    // Trigger an audit entry by querying the audit log
    await container.logs.find({ pagination: { limit: 1 } }).catch(() => {})

    await new Promise((r) => setTimeout(r, 1500))

    ac.abort()
    try { await streamPromise } catch {}

    if (streamErr) throw streamErr

    expect(received.length).toBeGreaterThan(0)
    const parsed = JSON.parse(received[0])
    expect(parsed).toHaveProperty("data")
  }, 15000)

  it("rejects unauthenticated stream requests", async () => {
    const anon = new HestiaClient({ baseUrl: BASE_URL })
    const errors: Error[] = []
    const ac = new AbortController()

    const streamPromise = anon.client.openStream(
      "/system/audit/log/stream",
      {
        onMessage: () => {},
        onError: (err) => { errors.push(err) },
        onClose: () => {},
      },
      { signal: ac.signal },
    )

    await new Promise((r) => setTimeout(r, 1500))

    ac.abort()
    try { await streamPromise } catch {}

    expect(errors.length).toBeGreaterThanOrEqual(1)
  }, 10000)
})
