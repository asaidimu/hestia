import { SchemaDefinition } from "@asaidimu/utils-schema"
import type { Document } from "../core/types"

export interface CollectionMeta {
  name: string
  schema: SchemaDefinition
  created: string
  updated: string
}

export type CollectionDocument = Document<{ schema: any }>
