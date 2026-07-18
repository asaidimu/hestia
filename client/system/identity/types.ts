import type { Document } from "../../core/types"

export interface UserData {
  email: string
  name: string
  verified: boolean
  permissions: string[]
  deleted?: string | null
}

export type UserIdentity = Document<UserData>

export interface CreateUserRequest {
  email: string
  password: string
  name: string
  permissions?: string[]
  verified?: boolean
}

export interface UpdateUserRequest {
  name?: string
  email?: string
  permissions?: string[]
  verified?: boolean
}

export interface ChangePasswordRequest {
  current: string
  new: string
}
