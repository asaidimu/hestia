# administrator → root rename (pure aesthetic)

**Intent:** top-level managers object to holding the `administrator` role;
`root` is a straight rename with identical power. Defaults, seeds, tests,
client, and docs all moved. No migration: alpha, all installs resettable.

- [*] Rule definitions + default
  - **Files:** `core/system/policies/rulecompiler.go` (GoDefaultRules),
    `core/system/policies/defaults.go` (seed CEL), `core/system/gen_features.go`
    (empty-RuleKey fallback).
- [*] Bindings via annotations + regen
  - 105 × `rule="administrator"` → `rule="root"` across all 14 service.go,
    `service generate --all` clean (rule-key-only diffs). Generator
    testdata/goldens deliberately untouched (synthetic fixtures).
  - Render fallback for empty rule: `authenticated` → `root`
    (`cmd/hestia/core/gen/render.go`).
- [*] Hand-written spots
  - Blob namespace op bindings, collections document-op bindings, seed
    admin permissions (`root`, display name stays "System Administrator"),
    updates admin-user lookup, test-server claims, annotate comment.
- [*] Tests/client/docs
  - All Go test scopes/expressions/prose; TS tests, mock seed/auth/rules,
    mock comments; docs + skill references. Explicit per-operation
    `authenticated` bindings left as-is (intentional choices).
- [*] Verified
  - Build, vet, Go suite (0 failures), tsc, client suite (293 passed),
    fresh-boot live: admin holds `['root']`, rules are
    public/authenticated/password_reset/root, policies bind root.
