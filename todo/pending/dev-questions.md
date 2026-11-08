# 101 Questions Before Building a Startup on hestia

Senior-developer due diligence for adopting hestia as the platform foundation.
Each question is a probe into whether hestia can carry a production product —
not just a demo. Organized by risk surface. **Bold = the ones that would
block a go/no-go decision for most teams.**

---

## A. Architecture & mental model

1. What happens when a message handler panics — does the recovery link catch it, or does the whole server die?
2. Is the `module:feature:scope:action` naming scheme a hard contract, or can message names be arbitrary?
3. What is the message **processing guarantee** — at-most-once, at-least-once, exactly-once? Is there a retry/redelivery mechanism?
4. How do long-running tasks work — is there async dispatch, background workers, or is everything request-scoped?
5. Is there a shared in-process bus, or can messages be routed across nodes/processes?
6. What happens if a module's `Setup` fails — is the app rolled back cleanly, or left half-initialized?
7. How are module `Dependencies()` enforced — at registration, at runtime, or not at all?
8. Is the dispatcher chain order guaranteed, or is it incidental to how `DispatcherChain` is built?
9. What is the memory model for pooled `Result`s — can a pooled document escape its lifetime and corrupt another request?
10. Is there a bounded-concurrency/backpressure mechanism, or can a flood of requests exhaust memory?
11. **Can two hestia apps in one process coexist, or is there global/package-level state (ProjectName, singletons) that prevents it?**
12. What is the actual per-request overhead of the full dispatcher chain (bootstrap→secure→...→audit)?

## B. Security, authn & authz

13. **Is the security model defense-in-depth or a single `SecureDispatcher` link — could a custom module accidentally bypass it?**
14. How is `Internal: true` enforced — is it a routing flag or an actual security boundary?
15. Can a module register a message that *shadows* a built-in `system:*` message and hijack auth routes?
16. What exactly is in a JWT — claims, scopes, expiry? Can tokens be revoked server-side?
17. How are session tokens hashed/signed — are leaked DB credentials sufficient to forge sessions?
18. Is there CSRF protection for cookie-based sessions, or is it assumed SameSite is enough?
19. **How granular is authorization — resource-level or just operation-level? Can rules express "own document only"?**
20. Are policy rules evaluated per-request, or cached with a TTL that can serve stale permissions?
21. Can rate limits and throttling be bypassed by switching API keys, or are they user/IP-scoped?
22. Is the bootstrap flow a security risk — is the ephemeral key logged, leaked in responses, or reversible?
23. What happens to audit logs on failure — are denied requests audited, and can audit itself be tampered with?
24. Are secrets (session secret, SMTP creds, API key hashes) sanitized from every log path — startup banner included?
25. Is `ForceBootstrapped` a foot-gun — what does it skip, and can it create an admin with a known password?
26. How are tenants isolated — is tenant scoping enforced in the chain for *every* message, or opt-in per feature?
27. Is the API key `validate` message itself rate-limited or abuseable as an oracle?
28. Does password auth protect against timing attacks / account enumeration?

## C. Persistence & data layer

29. **What is the consistency model — is SQLite WAL the only backend, and is it safe under concurrent writers across processes?**
30. Can I define composite/unique constraints beyond `unique: true` on single fields?
31. How are migrations handled for *production* data — destructive changes, rollback, or squash-only?
32. Is there a transaction story — can a handler span multiple document writes atomically?
33. What happens on schema change — are old documents migrated in place, lazily, or left stale?
34. Is there a query DSL for anything beyond simple `where/eq` — joins, aggregates, pagination cursors, text search?
35. Can I use my existing relational schema, or am I locked into document-style collections?
36. What is the blob storage story — filesystem only? S3-compatible backends? Data durability across restarts?
37. Are blobs backed up/restored together with the DB, or is there a split-brain risk?
38. **What happens at scale — at what document count / write rate does the embedded SQLite + in-process model break?**
39. Is `PersistenceFactory` a real escape hatch, or do features depend on anansi-specific assumptions?
40. How are `_metadata_` / `_id_` fields managed — can I index on them, and do they collide with my domain fields?

## D. Extensibility & customization

41. **Can I write a custom dispatcher link that runs *before* auth/audit, or is the chain fixed at boot?**
42. Is there a decorator/observer pattern, or is message handling strictly `handler` functions?
43. Can I intercept and rewrite messages/inputs between chain links?
44. **How do I add a completely custom HTTP endpoint (webhook, graphql, websocket) that bypasses the message system?**
45. Is `BuildInterfaces` sufficient, or are there framework assumptions (cookie handling, routing) that leak into custom interfaces?
46. Can I swap auth entirely — e.g. OIDC/SSO, custom credential providers, hardware tokens?
47. Is there an events/notifications pub-sub I can hook for domain events, or only the built-in notifier?
48. Can I add custom CEL functions/rules, or am I limited to the built-in rule set?
49. Is there a plugin system, or is every extension a Go package compiled in?
50. **Can I extend the TypeScript SDK with my own module's methods, or is it generated/static?**
51. How do I hook into the request lifecycle (cors, logging, tracing, request-id) beyond the provided middleware?
52. Can a module expose its own seed data that runs *before* other modules' seeds?

## E. Configuration & operations

53. Is configuration validated at boot (typos, invalid enum values) or silently ignored?
54. Can config be hot-reloaded, or is a restart required for every change?
55. What is the deployment story — single binary? Docker? Multi-node? Graceful shutdown?
56. Is there a health endpoint that reflects *module* health (the `Health` method) over HTTP?
57. **Can I run migrations as part of CI/CD, or only at app boot?**
58. Is there a maintenance mode / draining of in-flight requests on shutdown?
59. How is logging structured — is it JSON, is there a request-id correlation end-to-end?
60. Is there structured metrics/OpenTelemetry support, or is observability log-only?
61. Can I set per-feature log levels, or is it global?
62. What is the actual disk footprint — DB, blobs, event bus, logs — and can they be moved independently?
63. Is there a backup/restore procedure documented for the whole data dir?

## F. Performance & scalability

64. What is the baseline throughput/latency for a simple message round-trip through the full chain?
65. How does memory scale with concurrent sessions and pooled documents — any leaks under load?
66. Is there a benchmark suite, or are numbers anecdotal?
67. Can the HTTP layer be scaled across processes (shared DB), or is it single-process-only?
68. What is the cost of the audit link on every request — is audit async or synchronous?
69. How are blob uploads handled for large files — memory-buffered or streamed to disk?
70. Is there a connection pool limit for the embedded DB, and what happens at saturation?
71. **Is the event bus (Pebble) durable — do notifications/schedules survive a crash mid-publish?**

## G. Reliability & correctness

72. What happens if SMTP is down — do notifications queue, retry, or silently drop?
73. Are scheduled jobs persisted and recovered across restarts, or do they reschedule from scratch?
74. Is there a dead-letter / failed-job story for schedules?
75. **What is the upgrade path across hestia versions — is there a data migration + API compatibility contract?**
76. Is there a rollback strategy if a hestia upgrade breaks the app?
77. How are concurrent updates to the same document handled — last-write-wins, optimistic locking, or conflict errors?
78. What happens to in-flight requests during `app.Shutdown` — graceful drain or abrupt kill?
79. Is `Close()` idempotent, and does it flush logs/blobs/queues?
80. Are there known race conditions in the generated singleton models (`Init` + concurrent access)?
81. What is the test coverage philosophy — is the framework itself tested, or is that my problem?

## H. Multi-tenancy & data isolation

82. Is tenancy a first-class concept (tenant-scoped collections) or a convention enforced by the tenant link?
83. Can a tenant see another tenant's data through a cross-tenant query, by default?
84. Are blobs and schedules tenant-scoped too, or only documents?
85. Can I create tenants programmatically, and does it require a super-admin path?
86. How do policies interact with tenants — are rules per-tenant or global?

## I. SDK & API surface

87. Is the TypeScript SDK complete enough for production, or does it lag the Go API?
88. Does the SDK handle token refresh, retries, and error typing properly?
89. Is there a server-generated OpenAPI/Swagger spec, or only the docs endpoint?
90. Can the SDK work offline / for mobile, or is it browser/desktop only?
91. Is the generated route-to-SDK mapping stable across versions?

## J. Community, licensing & risk

92. **What is the license, and are all dependencies (go-anansi, go-iam, go-events, pebble) compatible with a commercial closed-source product?**
93. What happens if the maintainers abandon hestia — can the repo be forked and maintained? (How much is in-tree vs. external modules?)
94. Is there a changelog/semver policy, or does "alpha" mean anything goes?
95. How responsive is the project to issues/PRs — is it a solo project or a community?
96. What is the hiring/onboarding cost — how easy is it to hire engineers who know this stack?
97. Are there real-world production deployments, or is it pre-production?
98. What is the documentation quality for the *hard* parts (auth internals, security model, persistence)?
99. Is there a stable public API surface, or do imports like `core/internal/*` leak into extension code?
100. What are the known limitations the maintainers admit to — and are they showstoppers?
101. **If this is my only persistence layer, how much of my app logic becomes hestia-specific and hard to migrate away later (vendor lock-in)?**
