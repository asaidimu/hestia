import { describe, expect, it, afterAll } from "vitest"
import { makeClient } from "../../tests/helpers"

// E2E against the test server. `create` is excluded: every registered
// operation already has a seeded policy (POLICY_ALREADY_EXISTS).
const OP = "system:apikeys:key:create"

describe("HestiaPolicies — E2E", () => {
  const container = makeClient()

  afterAll(async () => {
    // restore the seeded state in case a test left it disabled
    await container.policies.setEnabled(OP, true).catch(() => {})
  })

  it("queries the operation-policy collection", async () => {
    const page = await container.policies.query({})
    expect(page.data.length).toBeGreaterThan(0)
    expect(page.data.some((p) => p.operation === OP)).toBe(true)
  })

  it("lists policies", async () => {
    const page = await container.policies.list()
    expect(page.data.length).toBeGreaterThan(0)
    expect(page.data.some((p) => p.operation === "system:auth:session:create")).toBe(true)
  })

  it("disables and re-enables a policy", async () => {
    const disabled = await container.policies.setEnabled(OP, false)
    expect(disabled.operation).toBe(OP)
    expect(disabled.enabled).toBe(false)

    const enabled = await container.policies.setEnabled(OP, true)
    expect(enabled.operation).toBe(OP)
    expect(enabled.enabled).toBe(true)
  })

  it("reads a policy by operation", async () => {
    const doc = await container.policies.read(OP)
    expect(doc).toBeDefined()
    expect(doc!.operation).toBe(OP)
  })
})