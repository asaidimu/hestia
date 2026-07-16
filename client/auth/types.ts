import { UserIdentity } from "../system/identity/types"

export interface TokenPair {
  access: string
  refresh: string
  type: string
  validity: number
}

export interface LoginResult {
  token: TokenPair
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

export interface RefreshRequest {
  refresh_token: string
}

export interface BootstrapPasswordRequest {
  password: string
  email: string
}
