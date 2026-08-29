# Hedwig Cleanup

- [ ] Remove hedwig/ui/src/.../flows/components/editor/api.ts
  - **Context:** Was a demo shim fetching registry/handles from localhost:3001 (hermes direct). Now that hestia exposes both via `client.workflows.getRegistry()` and `client.workflows.fetchHandles()`, this file is dead code.
  - **Migration:** Replace all `import { ... } from "./api"` with direct hestia client calls. The `RegistryProvider` should use `client.workflows.getRegistry()` instead of `fetchNodeRegistry()`. Handles are fetched via `client.workflows.fetchHandles()` and accessed synchronously via `getCachedHandles()`.
  - **Files:** hedwig/ui/src/interface/platform/flows/components/editor/api.ts

- [ ] Update RegistryProvider to use hestia client
  - **Context:** Currently calls `fetchNodeRegistry()` from the deprecated api.ts. Should call `client.workflows.getRegistry()` instead.
  - **Files:** hedwig/ui/src/interface/platform/flows/components/editor/registry-context.tsx

- [ ] Update handle fetching to use hestia client
  - **Context:** Currently calls `fetchNodeHandles()` from the deprecated api.ts. Should call `client.workflows.fetchHandles()` instead, with `getCachedHandles()` for synchronous access during rendering.
  - **Files:** hedwig/ui/src/interface/platform/flows/components/editor/api.ts (consumers), view-dispatcher.tsx
