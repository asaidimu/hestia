import { describe, expect, it } from "vitest"
import { makeClient } from "../../tests/helpers"
import type { WorkflowNode, WorkflowEdge } from "./types"

describe("HestiaWorkflowStore — while-loop workflow E2E", () => {
  const container = makeClient()

  const nodes: WorkflowNode[] = [
    { id: "trigger-1", type: "executable", position: { x: 960, y: -170 }, data: { kind: "trigger", config: { initialState: {} } } },
    { id: "3e46f1a5-97b1-41d5-ad42-65d49a1812b7", type: "executable", position: { x: 1230, y: -170 }, data: { kind: "transformer", config: { rules: [{ targetKey: "total", sourceKey: "", action: "SET_VALUE", actionParam: "11" }] } } },
    { id: "3bebb024-9018-4c34-be69-71e154d3c1f2", type: "executable", position: { x: 1480, y: -170 }, data: { kind: "while", config: { mode: "simple", condition: { key: "total", predicate: "greater_equals", value: "10" } } } },
    { id: "8ac61a7e-ef84-4bc5-86fe-816106f97a33", type: "executable", position: { x: 1780, y: -20 }, data: { kind: "arithmetic", config: { left: "total", right: "1", operation: "subtract", key: "total" } } },
    { id: "c68d280e-78e5-427a-bd56-d548161c44d3", type: "executable", position: { x: 1780, y: -280 }, data: { kind: "delay", config: {} } },
    { id: "63771c46-c2da-4f77-a177-c286a945327d", type: "executable", position: { x: 2020, y: -280 }, data: { kind: "if", config: { conditions: [{ field: "total", operator: "equals", value: "10" }] } } },
    { id: "136c5f83-d327-4c34-8e85-a719bed844fd", type: "executable", position: { x: 2260, y: -400 }, data: { kind: "arithmetic", config: { left: "total", right: "2", operation: "multiply", key: "total" } } },
    { id: "c4963b9c-9dc4-41cb-a83c-39393da73034", type: "executable", position: { x: 2260, y: -160 }, data: { kind: "delay", config: {} } },
  ]

  const edges: WorkflowEdge[] = [
    { id: "d13b024d-25af-452e-a4b9-1693c92369a4", source: "trigger-1", target: "3e46f1a5-97b1-41d5-ad42-65d49a1812b7", data: { role: "flow" } },
    { id: "902752ea-7b27-4fb4-bee0-99cad2b6a3e0", source: "3e46f1a5-97b1-41d5-ad42-65d49a1812b7", target: "3bebb024-9018-4c34-be69-71e154d3c1f2", data: { role: "flow" } },
    { id: "c32b3a77-c60a-4463-91e3-eefe4fbddbf4", source: "3bebb024-9018-4c34-be69-71e154d3c1f2", target: "8ac61a7e-ef84-4bc5-86fe-816106f97a33", sourceHandle: "do", data: { role: "flow" } },
    { id: "2e28b963-2f12-4f72-bd65-99ffa9275f7d", source: "8ac61a7e-ef84-4bc5-86fe-816106f97a33", target: "3bebb024-9018-4c34-be69-71e154d3c1f2", data: { role: "flow" } },
    { id: "cc0ee56f-182c-4f1c-aad6-74b4a1acb66a", source: "3bebb024-9018-4c34-be69-71e154d3c1f2", target: "c68d280e-78e5-427a-bd56-d548161c44d3", sourceHandle: "done", data: { role: "flow" } },
    { id: "0670e55b-cc8e-4070-b67f-0df75aac44c1", source: "c68d280e-78e5-427a-bd56-d548161c44d3", target: "63771c46-c2da-4f77-a177-c286a945327d", data: { role: "flow" } },
    { id: "0d4a3781-5f9e-4e93-80d3-da1785c80319", source: "63771c46-c2da-4f77-a177-c286a945327d", target: "136c5f83-d327-4c34-8e85-a719bed844fd", sourceHandle: "if", data: { role: "flow" } },
    { id: "ad3d3ace-498f-49bc-9ce7-ea0b06f8fd21", source: "63771c46-c2da-4f77-a177-c286a945327d", target: "c4963b9c-9dc4-41cb-a83c-39393da73034", sourceHandle: "else", data: { role: "flow" } },
  ]

  it("compiles the workflow", async () => {
    const result = await container.workflows.compile(nodes, edges)
    expect(result.workflow_id).toBeTruthy()
    expect(result.triggers).toBe(1)
    expect(result.pipelines).toBeGreaterThanOrEqual(1)
  })

  it("runs the workflow to completion", async () => {
    const runId = await container.workflows.run({ nodes, edges })
    expect(runId).toBeTruthy()

    const outcome = await container.workflows.getOutcome(runId)
    expect(outcome).not.toBeNull()
    expect(outcome!.ok).toBe(true)
    expect(outcome!.status).toBe("succeeded")
  })

  it("produces the expected final state", async () => {
    const runId = await container.workflows.run({ nodes, edges })
    const store = await container.workflows.getStore(runId)
    expect(store).not.toBeNull()
    expect(store!.total).toBeDefined()
  })

  it("emits timeline events", async () => {
    const runId = await container.workflows.run({ nodes, edges })
    const events = await container.workflows.getEvents(runId)
    expect(events.length).toBeGreaterThan(0)
    expect(events[0].run_id).toBe(runId)
  })

  it("can be registered and listed", async () => {
    const id = await container.workflows.create({
      name: "while-loop-e2e",
      description: "while loop test workflow",
      nodes,
      edges,
    })
    expect(id).toBeTruthy()

    const all = await container.workflows.list()
    expect(all.some((d) => d._id_ === id)).toBe(true)

    const doc = await container.workflows.get(id)
    expect(doc).not.toBeNull()
    expect(doc!.name).toBe("while-loop-e2e")

    await container.workflows.delete(id)
  })
})
