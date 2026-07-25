import { describe, expect, it, vi, beforeEach, afterEach } from "vitest"
import { WailsTransport } from "../core/wails-transport"
import { HttpTransport, type IdentityProvider } from "../core/client"
import { SystemError } from "@asaidimu/utils-error"

function makeProvider(): IdentityProvider {
  return {
    identity: () => null,
    setIdentity: vi.fn(),
    clear: vi.fn(),
  }
}

function mockDispatch(result: { status: number; data?: unknown }) {
  const dispatch = vi.fn().mockResolvedValue(result)
  ;(globalThis as any).window = {
    go: {
      hestia: {
        Api: { Dispatch: dispatch },
      },
    },
  }
  return dispatch
}

function mockNoGo() {
  ;(globalThis as any).window = {}
}

describe("WailsTransport notifyAuthStateChange", () => {
  beforeEach(() => {
    mockNoGo()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    delete (globalThis as any).window
  })

  it("calls onUnauthorized on 401 by default", async () => {
    mockDispatch({ status: 401 })
    const onUnauthorized = vi.fn()
    const t = new WailsTransport({
      pkg: "hestia",
      struct: "Api",
      baseUrl: "http://test.local",
      apiPrefix: "/api",
      identityProvider: makeProvider(),
      onUnauthorized,
    })
    await t.ready()

    await expect(t.dispatch("system:core:heartbeat")).rejects.toThrow(SystemError)

    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  it("suppresses onUnauthorized when notifyAuthStateChange is false", async () => {
    mockDispatch({ status: 401 })
    const onUnauthorized = vi.fn()
    const t = new WailsTransport({
      pkg: "hestia",
      struct: "Api",
      baseUrl: "http://test.local",
      apiPrefix: "/api",
      identityProvider: makeProvider(),
      onUnauthorized,
    })
    await t.ready()

    await expect(
      t.dispatch("system:core:heartbeat", { notifyAuthStateChange: false }),
    ).rejects.toThrow(SystemError)

    expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it("still calls onUnauthorized on non-401 errors (default)", async () => {
    mockDispatch({ status: 500, data: { error: { code: "INTERNAL", message: "fail" } } })
    const onUnauthorized = vi.fn()
    const t = new WailsTransport({
      pkg: "hestia",
      struct: "Api",
      identityProvider: makeProvider(),
      onUnauthorized,
    })
    await t.ready()

    await expect(t.dispatch("system:core:heartbeat")).rejects.toThrow(SystemError)

    expect(onUnauthorized).not.toHaveBeenCalled()
  })
})
