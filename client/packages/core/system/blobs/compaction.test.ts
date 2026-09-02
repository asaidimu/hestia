import { describe, expect, it, afterAll } from "vitest"
import { BASE_URL } from "../../tests/helpers"

const API = `${BASE_URL}/api`

function uniqueId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

describe.skip("Namespace compaction — HTTP E2E", () => {
  const ns = uniqueId("compact-test")
  const blobKey = uniqueId("file")

  afterAll(async () => {
    await fetch(`${API}/system/blobs/namespace/delete/${ns}`, { method: "DELETE" })
  })

  it("uploads a file, deletes it, then compacts", async () => {
    // 1. Create namespace
    const createNs = await fetch(`${API}/system/blobs/namespace/create/${ns}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ display_name: "Compaction Test" }),
    })
    expect(createNs.ok).toBe(true)

    // 2. Upload a file (direct single-shot upload)
    const fileContent = "Hello, compaction test!"
    const uploadRes = await fetch(`${API}/system/blobs/blob/upload/${ns}/${blobKey}`, {
      method: "POST",
      headers: { "Content-Type": "text/plain" },
      body: fileContent,
    })
    expect(uploadRes.ok).toBe(true)
    const uploadBody = await uploadRes.json()
    expect(uploadBody.data.key).toBe(blobKey)
    expect(uploadBody.data.size).toBe(fileContent.length)

    // 3. Delete the blob
    const deleteRes = await fetch(`${API}/system/blobs/blob/delete/${ns}/${blobKey}`, {
      method: "DELETE",
    })
    expect(deleteRes.ok).toBe(true)

    // 4. Compact the namespace
    const compactRes = await fetch(`${API}/system/blobs/namespace/compact/${ns}`, {
      method: "POST",
    })
    expect(compactRes.ok).toBe(true)
    const compactBody = await compactRes.json()
    expect(compactBody.data).toBeDefined()
    expect(typeof compactBody.data.blobs_removed).toBe("number")
    expect(typeof compactBody.data.chunks_removed).toBe("number")
    expect(typeof compactBody.data.bytes_freed).toBe("number")
    expect(typeof compactBody.data.segments_compacted).toBe("number")
  })
})
