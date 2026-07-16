// Core
export { HestiaNetworkClient, HestiaResponse } from "./core/client"
export type { IdentityProvider  } from "./core/client"
export { HestiaCollection } from "./core/collection"
export { toSystemError } from "./core/errors"

// Generic types
export type { Document, Page, PagedData, PaginationInfo, StoreEvent } from "./core/types"

export type { PageOptions } from "./core/pager"

// Auth
export { HestiaAuth } from "./auth/store"
export type { TokenPair, LoginResult, ServerHealth, LoginRequest, RegisterRequest } from "./auth/types"

// System: identity
export { HestiaUsers } from "./system/identity/store"
export type {
  UserData,
  UserIdentity,
  CreateUserRequest,
  UpdateUserRequest,
} from "./system/identity/types"

// System: api-keys
export { HestiaKeyStore } from "./system/api-keys/store"
export type { APIKey, APIKeyWithSecret, CreateKeyRequest, UpdateKeyRequest } from "./system/api-keys/types"

// System: policies
export { HestiaPolicies } from "./system/policies/store"
export type {
  PolicyOperation,
  PolicyRule,
  UpsertOperationRequest,
  UpsertRuleRequest,
  ValidateRuleRequest,
  ValidateRuleResult,
  ReloadResult,
} from "./system/policies/types"

// System: logs
export { HestiaLogs } from "./system/logs/store"
export type { AuditEntry, LogFilter } from "./system/logs/types"

// Blobs
export { HestiaBlobClient, BlobNamespace } from "./blobs/store"
export type {
  NamespaceInfo,
  BlobMeta,
  BlobDocument as BlobDoc,
  ListBlobsRequest,
  CreateNamespaceRequest,
} from "./blobs/types"

// Collections
export { HestiaCollections } from "./collections/store"
export type { CollectionMeta } from "./collections/types"

// Container
export { HestiaClient } from "./container"

export * from "./utils"
