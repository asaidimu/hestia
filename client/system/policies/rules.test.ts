import { describe, expect, it, beforeAll, afterAll } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"

describe("HestiaRules — E2E", () => {
  const container = makeClient()
  const ruleName = uniqueId("e2e_rule")

  afterAll(async () => {
    await container.rules.delete(ruleName).catch(() => {})
  })

  it("lists policy rules", async () => {
    const page = await container.rules.list()
    expect(page.data.length).toBeGreaterThan(0)
    expect(page.data.some((r) => r.name === "administrator")).toBe(true)
  })

  it("gets a built-in rule by name", async () => {
    const rule = await container.rules.read("administrator")
    expect(rule).toBeDefined()
    expect(rule!._id_).toBeTruthy()
    expect(rule!.name).toBe("administrator")
  })

  it("creates a rule", async () => {
    const rule = await container.rules.create({
      data: { expression: "identity != null" },
      options: ruleName,
    })
    expect(rule).toBeDefined()
    expect(rule!._id_).toBeTruthy()
    expect(rule!.name).toBe(ruleName)
  })

  it("gets the created rule by name", async () => {
    const rule = await container.rules.read(ruleName)
    expect(rule).toBeDefined()
    expect(rule!.name).toBe(ruleName)
  })

  it("updates a rule", async () => {
    const updated = await container.rules.update({
      data: { expression: "false" },
      options: ruleName,
    })
    expect(updated).toBeDefined()
    expect(updated!.expression).toBe("false")
  })

  it("deletes a rule", async () => {
    await container.rules.delete(ruleName)
  })

  it("validates a rule expression", async () => {
    const result = await container.rules.validate({ rule: "identity.is_admin == true" })
    expect(result.valid).toBe(true)
  })

  it("reloads policies", async () => {
    const result = await container.rules.reload()
    expect(typeof result.rules).toBe("number")
    expect(typeof result.operations).toBe("number")
  })
})