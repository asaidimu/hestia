# Client HestiaDataView (view counterpart to HestiaCollection)

**Intent:** The TypeScript SDK (`client/packages/core`) needs a `HestiaDataView`
class mirroring `HestiaCollection` for the server-side view messages shipped
earlier (`system:collections:view:create|refresh`, queryable via
`system:collections:document:query`). Creation is a static on the class itself
(`HestiaDataView.create(...)`); `read(id)` is kept since `document:get` works
on views. No `view:delete` affordance (blocked upstream — virtual views are
undroppable; see `todo/anansi_sqlite_views_fts.md`).

**Reference files:**
- `client/packages/core/core/collection.ts` (shape to mirror),
  `core/collection.test.ts` (E2E pattern), `core/client.ts` (RouteName-typed
  dispatch), `core/routes.gen.ts` (STALE copy — sync from generator),
  `system/collections/store.ts` (`HestiaCollections`), `system/collections/types.ts`
  (`CollectionMeta`), `container.ts` (`collection<T>` accessor),
  `client/packages/mock/src/routes.ts` (hand-maintained mock table),
  `client/packages/core/index.ts` (exports), `client/packages/core/README.md`.
- Server messages: `core/system/collections/{views,service}.go`;
  generator: `cmd/gen-routes/main.go` (writes `client/core/routes.gen.ts`).

- [x] Fix gen-routes to write the package route table + update CI drift path
  - **Context:** `Transport.dispatch` is typed by `RouteName` from
    `client/packages/core/core/routes.gen.ts`, but `cmd/gen-routes` writes
    `client/core/routes.gen.ts` (git-untracked, nothing syncs them). Without
    the fix, `dispatch("system:collections:view:create")` does not typecheck.
  - **Details:** Change `outPath` in `cmd/gen-routes/main.go` to
    `client/packages/core/core/routes.gen.ts`; update the drift check in
    `.github/workflows/test.yaml` to the same path; remove the stale
    `client/core/` dir.
  - **Files:** `cmd/gen-routes/main.go`, `.github/workflows/test.yaml`.
- [*] Add DataViewMeta type + HestiaDataView class
  - **Context:** Mirror `core/collection.ts` (`DocumentStore`, paged
    controller, envelope parsing). Views are read-only server-side.
  - **Details:** `DataViewMeta { name, materialized, target }` in
    `system/collections/types.ts`. New `core/data-view.ts` with
    `HestiaDataView<T>`: `static create(client, {name, query, materialized})`
    → `view:create`; `find/list/page` → `document:query`; `read(id)` →
    `document:get`; `refresh()` → `view:refresh`; document
    `create/update/delete` + `upload/stream/subscribe/notify` throw. Reuse
    `QueryDSL<T>` from `@asaidimu/query` for the stored-query payload.
  - **Files:** `client/packages/core/system/collections/types.ts`,
    `client/packages/core/core/data-view.ts`.
- [*] Wire accessors + exports + README
  - **Details:** `HestiaCollections.view<T>(name)` accessor,
    `HestiaClient.view<T>(name)` in `container.ts` (mirrors
    `collection<T>`), `export { HestiaDataView }` in `index.ts`, README
    feature row next to the `HestiaCollection<T>` line.
  - **Files:** `system/collections/store.ts`, `container.ts`, `index.ts`,
    `packages/core/README.md`.
- [*] Add view routes to the mock transport table
  - **Details:** `view:create` (POST
    `/system/collections/view/create/{name}`) + `view:refresh` (PATCH
    `.../view/refresh/{name}`), matching generator output format.
  - **Files:** `client/packages/mock/src/routes.ts`.
- [*] Add data-view E2E test and run the client suite against :8070
  - **Result:** full client suite green — 23 files, 283 tests passed
    (`bunx vitest --run --fileParallelism=false` against :8070).
- [*] Document proxy with id()/metadata() on every document
  - **Context:** follow-up request — `Document<T>` is envelope
    (`_id_`/`_metadata_`) + payload, so a `Proxy` can attach methods
    transparently.
  - **Details:** new `core/document.ts` (`proxiedDocument`, `proxiedPage`,
    `ProxiedDocument<T>`, `DocumentMethods`); methods are non-own properties
    so `Object.keys`/spread/`JSON.stringify` are unaffected; a payload field
    literally named `id`/`metadata` wins over the method. Wired into
    `HestiaCollection` (find/read/create/update) and `HestiaDataView`
    (find/read/refresh) return types; exported `DocumentMetadata` from
    `core/types.ts`. Unit tests in `core/document.test.ts` (5/5, no server).
  - **Files:** `client/packages/core/core/document.ts`,
    `document.test.ts`, `collection.ts`, `data-view.ts`, `core/types.ts`,
    `index.ts`.
  - **Context:** Mirror `core/collection.test.ts` via `tests/helpers.ts`
    (`makeClient`, `collectionSchema`, `uniqueId`); test-server on :8070
    must be running (`make test-client` / `bunx vitest`).
  - **Details:** create base collection + docs → `HestiaDataView.create`
    (virtual) → `find` composes stored filter → doc write throws →
    `refresh` on virtual throws → materialized snapshot stale → `refresh`
    → fresh. Then `bunx vitest --run` for the package (incl. auto-expanded
    `routes.test.ts` coverage of the new routes).
  - **Files:** `client/packages/core/core/data-view.test.ts`.

- [*] Fix #5ag9xj (explicit update addressing) + strip envelope keys + update_many
  - **Context:** devnote #5ag9xj — `update({data, options})` smuggled the id
    through `options`. First attempt (`{id, data}`, options dropped) was
    rejected: filter-based updates need a target channel. Resolved per
    decisions: exactly-one-of id/filter, generic collections only.
  - **Details:** `DocumentStore.update` contract is now
    `{data, id?, filter?, options?}` with an XOR guard; all 10 implementers
    migrated (id-only stores throw on filter); server gained
    `system:collections:document:update_many` (filter-only route — the
    derived `document:update` route embeds mandatory `/{doc_id}`, so a
    separate message was required) sharing `applyDocumentUpdate` with the
    by-id handler; `strippedData()` helper removes `_id_`/`_metadata_` from
    every outbound document payload (collection/apikeys/users/rules/
    notifications/schedules/blobs-custom). E2E: filter update, guard
    rejection, forged-`_id_` stripped live. Full suites green (client 289,
    Go 32 pkgs). #5ag9xj comment removed + index re-synced; #qhx9r0
    (pager options) left open.
  - **Upstream follow-up:** physical-name truncation collision found via the
    snapshot E2E (deterministic 3-vs-2); recorded as devnote
    #physical-name-truncation-collide-496cab49 in go-anansi
    (`core/persistence/registry/utils.go:42`, P1). Client E2E uses short
    unique ids (`shortId`) as a workaround.
