import { Document } from "../../core/types"

export interface UserData {
  email: string
  name: string
  verified: boolean
  scopes: string[]
  deleted_at?: string | null
}

export type UserIdentity = Document<UserData>

export interface CreateUserRequest {
  email: string
  password: string
  name: string
  scopes?: string[]
  verified?: boolean
}

export interface UpdateUserRequest {
  name?: string
  email?: string
  scopes?: string[]
  verified?: boolean
}

export interface ChangePasswordRequest {
  current: string
  new: string
}
