import { describe, expect, it } from "vitest"
import { proxiedDocument, proxiedPage, strippedData } from "./document"
import type { Document } from "./types"

function rawDoc(extra: Record<string, any> = {}) {
  return {
    _id_: "doc-1",
    _metadata_: { checksum: "abc", created: "1", updated: "2", version: 3 },
    ...extra,
  } as Document<{ title: string } & Record<string, any>>
}

describe("proxiedDocument", () => {
  it("exposes id() and metadata() from the envelope", () => {
    const doc = proxiedDocument(rawDoc({ title: "hello" }))
    expect(doc.id()).toBe("doc-1")
    expect(doc.metadata()).toEqual({ checksum: "abc", created: "1", updated: "2", version: 3 })
    expect(doc.title).toBe("hello")
  })

  it("forwards writes to the underlying document", () => {
    const raw = rawDoc({ title: "hello" })
    const doc = proxiedDocument(raw)
    ;(doc as any).title = "bye"
    expect(raw.title).toBe("bye")
  })

  it("keeps serialisation free of the methods", () => {
    const doc = proxiedDocument(rawDoc({ title: "hello" }))
    expect(Object.keys(doc).sort()).toEqual(["_id_", "_metadata_", "title"])
    expect(JSON.parse(JSON.stringify(doc))).toEqual({
      _id_: "doc-1",
      _metadata_: { checksum: "abc", created: "1", updated: "2", version: 3 },
      title: "hello",
    })
    expect({ ...doc }.title).toBe("hello")
  })

  it("lets a payload field named id/metadata win over the method", () => {
    const doc = proxiedDocument(rawDoc({ id: "payload-id" }))
    expect((doc as any).id).toBe("payload-id")
    expect(doc.metadata()).toBeDefined()
  })

  it("proxiedPage maps every row", () => {
    const page = proxiedPage({
      data: [rawDoc({ title: "a" }), rawDoc({ title: "b" })],
      loading: false,
      page: { number: 1, size: 2, count: 2, total: 2, pages: 1 },
      error: null,
    })
    expect(page.data.map((d) => d.id())).toEqual(["doc-1", "doc-1"])
    expect(page.data[0]!.title).toBe("a")
  })
})

describe("strippedData", () => {
  it("removes _id_ and _metadata_ but keeps everything else", () => {
    expect(strippedData({ _id_: "x", _metadata_: {}, title: "hello", n: 1 })).toEqual({
      title: "hello",
      n: 1,
    })
  })

  it("does not mutate the input and passes through non-objects", () => {
    const input = { _id_: "x", title: "hello" }
    const out = strippedData(input)
    expect(out).not.toBe(input)
    expect(input._id_).toBe("x")
    expect(strippedData(null as any)).toBeNull()
    expect(strippedData("s" as any)).toBe("s")
  })
})
