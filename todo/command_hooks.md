# Command Hooks (pre/post) — UNREASONED CAPTURE

**Status:** captured for later reasoning. Nothing here is decided; the open
questions below are the point of the exercise. Do not implement from this file
without first resolving them with the maintainer.

- [ ] Design (then implement) pre- and post-command hooks for message dispatch
  - **Context:** Hestia dispatches every operation ("command") through the
    async-native dispatcher chain (`abstract.Dispatcher.Send(ctx, msg,
    onComplete)`; chain assembled in `core/system/module.go`
    `DispatcherChain()`: bootstrap → secure → ratelimit → throttle → tenant →
    blob → audit → LocalDispatcher). Cross-cutting concerns are implemented as
    `DispatcherLink`s that wrap *all* traffic. There is currently no way for a
    feature/service author to attach logic scoped to *specific* operations —
    something that runs before a named command executes (e.g. extra
    validation, idempotency checks, payload enrichment) or after it completes
    (e.g. projections, cache invalidation, follow-up notifications) without
    writing a full global chain link or burying it inside the handler.
  - **Details (rough shape to reason about):**
    - Candidate surface #1: registration-scoped hooks — fields on
      `abstract.MessageRegistration` (e.g. `PreHooks []Hook`,
      `PostHooks []Hook`) applied by the terminal or by an installation step;
      annotation vocabulary would need matching attributes
      (`cmd/hestia/core/annotate` + `gen/render.go` emitter), following the
      `fire_and_forget="true"` precedent.
    - Candidate surface #2: a dedicated hook link near the innermost chain
      position that consults a registry keyed by message name
      (`module:feature:scope:action`), so hooks can be added/removed at
      runtime without rebuilding registrations.
    - Hook signature candidates: reuse `abstract.MessageHandler`, or a narrower
      `func(ctx context.Context, msg Message) error` for pre-hooks; post-hooks
      may want the outcome — note the async contract: post logic could hang off
      the `CompletionFunc` continuation (fires exactly once, possibly long
      after acceptance) rather than running inline after `Send` returns.
  - **Open questions to resolve first:**
    1. Ordering guarantees relative to existing links (before secure? between
       audit and terminal? configurable?).
    2. Do pre-hook errors reject synchronously (transport sees the error) and
       does that interact correctly with `FireAndForget` registrations?
    3. Post-hook timing semantics: inline-on-completion goroutine vs
       fire-and-forget side dispatch (compare with how ThrottleDispatcher
       actions work in `core/runtime/throttle.go`).
    4. Are hooks part of the public module API (`SetupConfig`) or internal-only
       initially? Policy implications: hooks bypass nothing — they must not
       weaken the secure/audit links' guarantees.
    5. Annotation vocabulary shape if registration-scoped wins (list-valued
       attribute parsing doesn't exist yet in the annotate parser).
  - **Files likely touched:** `core/abstract/module.go`,
    `core/abstract/system_options.go`, `core/system/module.go`,
    `core/runtime/local-dispatcher.go` (or a new hook link),
    `cmd/hestia/core/annotate/annotate.go`, `cmd/hestia/core/gen/render.go`,
    plus both `testdata/users` fixtures when the vocabulary changes.
