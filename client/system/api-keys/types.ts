export interface APIKey {
  _id_: string
  name: string
  prefix: string
  scopes: string[]
  status: string
  expiry?: string | null
  environment?: string
  _metadata_: Record<string, unknown>
}

export interface APIKeyWithSecret extends APIKey {
  key: string
}

export interface CreateKeyRequest {
  name: string
  environment?: string
  scopes?: string[]
  expiry?: string
}

export interface UpdateKeyRequest {
  name?: string
  environment?: string
  scopes?: string[]
  status?: "active" | "revoked"
  expiry?: string
}
