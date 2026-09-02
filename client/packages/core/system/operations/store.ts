import { type Transport } from "../../core/client"
import type { Document, Page } from "../../core/types"
import type { Binding } from "./types"

function toDocument(b: Binding): Document<Binding> {
  const raw = b as Document<Binding>
  return {
    ...b,
    _id_: raw._id_ ?? b.name,
    _metadata_: raw._metadata_ ?? { checksum: "", created: "", updated: "", version: 1 },
  }
}

export class HestiaOperations {
  constructor(private client: Transport) {}

  async list(): Promise<Page<Binding>> {
    const res = await this.client.dispatch<{ data: { bindings: Binding[] } }>(
      "system:policies:binding:list",
    )
    const items = res.data?.data?.bindings ?? []
    return {
      data: items.map(o => toDocument(o)),
      loading: false,
      page: { number: 1, size: items.length, count: items.length, total: items.length, pages: 1 },
      error: null,
    }
  }

  async read(name: string): Promise<Document<Binding> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: Binding }>(
        "system:policies:binding:get",
        { arguments: { name } },
      )
      if (!res.data?.data) return undefined
      return toDocument(res.data.data)
    } catch (err: any) {
      if (err?.code === "SYNC-001-NF" || err?.code === "NOT_FOUND") return undefined
      throw err
    }
  }
}