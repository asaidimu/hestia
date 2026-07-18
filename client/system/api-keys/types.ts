export interface APIKey {
  _id_: string
  name: string
  prefix: string
  operations: string[]
  status: string
  expiry?: string | null
  environment?: string
  usage?: number
  last_used?: string | null
  _metadata_: Record<string, unknown>
}

export interface APIKeyWithSecret extends APIKey {
  key: string
}

export interface CreateKeyRequest {
  name: string
  environment?: string
  operations?: string[]
  expiry?: string
}

export interface UpdateKeyRequest {
  name?: string
  environment?: string
  operations?: string[]
  status?: "active" | "revoked"
  expiry?: string
}
