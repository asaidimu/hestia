import type { UserIdentity } from "../users/types"

export interface LoginResult {
  user: UserIdentity
  /** Absolute session expiry, unix seconds (omitted by older servers). */
  expires_at?: number
}

export interface ServerHealth {
  bootstrapped: boolean
  ok: boolean
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
}

export interface BootstrapPasswordRequest {
  password: string
  email: string
}
