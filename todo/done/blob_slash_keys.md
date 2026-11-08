# Blob keys with slashes don't route (`such/keys/are/allowed.txt`)

**Intent:** Blob keys may contain `/`, but the HTTP trie matched `{key}` to
exactly one path segment (and fasthttp normalizes `%2F`), so any slashed key
404d with "no matching route". Fixed with greedy catch-all route segments.
Verified 2026-09-10: Go 32 pkgs ok, client 290 passed, live 201→204→404 on
the exact reported URL shape.

**Design note:** greediness is transport metadata, not schema shape —
`Input.CatchAll` follows the `ResourceIDField`/`Streaming` precedent
(markers on the registration beside the schema, never in it).

- [*] Greedy `{name*}` segments in the HTTP trie
  - **Details:** `findOrCreate`/`lookup` support a trailing-`*` param that
    consumes all remaining segments joined with `/`; only valid terminally
    (panic otherwise). No unescaping — fasthttp normalization is the single
    decode step (verified empirically: `%2F` arrives decoded). Unit tests in
    `router_test.go`.
  - **Files:** `core/interface/http/router.go`, `router_test.go`.
- [*] `catchall` annotation attr → registration → derived route
  - **Details:** `Input.CatchAll string` (arg name); `Arguments()` marks the
    matching `ArgumentDefinition`; `DeriveRoute` emits `{key*}`; annotate
    parses `catchall="key"`; gen emitter writes `CatchAll:`. Added to the 6
    key-bearing blob messages (head/upload/download/delete/update/rename);
    regen blobs + route table.
  - **Files:** `core/abstract/module.go`, `core/runtime/route/route.go`,
    `cmd/hestia/core/annotate/annotate.go`, `cmd/hestia/core/gen/render.go`,
    `core/system/blobs/service.go`, generated registrations + routes.gen.ts.
- [*] Client + mock URL building/matching for catch-all
  - **Details:** `substituteArgs` (client.ts, mock transport.ts) encodes
    `{key*}` per-segment (preserving `/`); mock `resolvePath` matches
    greedy tail; mock route table entries gain `*`; routes.test.ts helpers
    made greedy-aware.
  - **Files:** `client/packages/core/core/client.ts`,
    `client/packages/mock/src/{transport,routes}.ts`,
    `client/packages/core/core/routes.test.ts`,
    `client/packages/core/system/blobs/blobs.test.ts` (slashed-key E2E).
