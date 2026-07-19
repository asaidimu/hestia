import type { UserIdentity } from "../system/identity/types"

export interface LoginResult {
  user: UserIdentity
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
