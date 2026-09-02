import { HestiaClient } from "../container"
import type { SchemaDefinition } from "@asaidimu/utils-schema"

export const BASE_URL = "http://localhost:8070"

export function makeClient(): HestiaClient {
  return new HestiaClient({ baseUrl: BASE_URL })
}

export function uniqueId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Builds a valid anansi collection schema for a dynamically-created collection. */
export function collectionSchema(name: string): SchemaDefinition {
  return {
    name,
    version: "1.0.0",
    fields: {
      "019f4065-0a3d-7ea1-bc46-bbaeed4bfd6d": { name: "_id_", type: "string", required: true, unique: true },
      "019f4065-0a3d-7ecf-a2eb-7af6e1fdd6f0": { name: "_metadata_", type: "object", schema: { id: "019f4065-0a3d-7ed7-8ab1-417acc881135" } },
      "019f9999-0000-7000-8000-000000000003": { name: "title", type: "string" },
    },
    schemas: {
      "019f4065-0a3d-7ed7-8ab1-417acc881135": {
        name: "_metadata_",
        fields: {
          "019f32a2-1eb3-7c39-885e-c3d545f981ac": { name: "version", type: "number", required: true },
          "019f32a2-1eb5-72a9-a0d6-086140f78a85": { name: "updated", type: "string", required: true },
          "019f32a2-1eb5-7440-b104-8d774438853a": { name: "checksum", type: "string", required: true },
          "019f32a2-1eb5-78b8-971d-ac164c938f2f": { name: "created", type: "string", required: true },
        },
      },
    },
  }
}