# Cache Audit Notes

## Issue #cache-20260821-001: All caches use unbounded sync.Map instead of anansi cache

**Status:** Open
**Priority:** P1
**Tags:** #correctness, #performance

### Problem

Multiple caches in the codebase use `sync.Map` which is unbounded and never evicts. The anansi framework provides a production-grade cache at `~/projects/go-anansi/core/cache` that should be used instead.

### Caches Found

1. **docValidators** (`dispatch/input.go:159`)
   - Caches: `*definition.DocumentValidator` by schema pointer
   - Risk: Unbounded growth with schema count
   - Impact: Memory leak in long-running applications

2. **fieldsCache** (`util/structmap.go:15`)
   - Caches: `[]fieldInfo` by `reflect.Type`
   - Risk: Low (`reflect.Type` is finite, compile-time known)
   - Impact: Minimal, but still unbounded

3. **inputPoolCache** (`testutil/inputdoc.go:13`)
   - Caches: `*document.DocumentPool` by schema name
   - Risk: Low (test utility only)
   - Impact: Test memory usage

4. **envelopeSchemas** (`transport_fasthttp.go` - proposed)
   - Will cache: `*document.DocumentPool` by response schema
   - Risk: Unbounded if not using anansi cache
   - Impact: Response latency, memory usage

### Anansi Cache Benefits

- Sharded for reduced contention
- Read-optimized (shared locks for reads)
- Bounded with CLOCK eviction
- TTL-aware per key
- Self-compacting
- Hash-randomized (security)
- Stats and monitoring

### Resolution

1. Replace `sync.Map` with anansi cache for all production caches
2. Configure appropriate `MaxEntries` and TTL for each cache
3. Add metrics to monitor cache hit rates
4. Consider using `NegativeTTL` for schema validation failures

### Impact

- Prevents memory leaks in long-running applications
- Provides predictable memory usage for IoT/HFT
- Enables cache hit rate monitoring
- Reduces contention under high concurrency
