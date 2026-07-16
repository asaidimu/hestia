import type { Document } from "../core/types"

export interface NamespaceInfo {
  id: string
  display_name: string
}

export interface BlobMeta {
  key: string
  namespace_id: string
  content_type: string
  size: number
  created_at: string
  updated_at?: string
  custom?: Record<string, any>
}

export type BlobDocument = Document<BlobMeta>

export interface ListBlobsRequest {
  prefix?: string
  limit?: number
}

export interface CreateNamespaceRequest {
  display_name?: string
  ns?: string
}
