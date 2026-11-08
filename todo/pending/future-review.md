# Future Correctness Review

## Context

This review focuses on correctness issues in the core framework, excluding core system features (system/* packages). The goal is to identify bugs, race conditions, resource leaks, and other correctness issues that could cause failures in production.

## Review Areas

### 1. Error Handling
- [ ] Check if errors are wrapped with `common.SystemError`
- [ ] Identify swallowed errors (err != nil but not handled)
- [ ] Verify error propagation in dispatch chain
- [ ] Check for proper error types in HTTP responses

**Files to examine:**
- `core/runtime/dispatch/*.go`
- `core/interface/http/*.go`
- `core/runtime/*.go`

### 2. Nil Safety
- [ ] Check for potential nil pointer dereferences
- [ ] Verify nil checks before type assertions
- [ ] Check nil maps/slices passed to functions
- [ ] Verify nil checks in HTTP handlers

**Files to examine:**
- `core/runtime/dispatch/dispatch.go`
- `core/interface/http/transport_fasthttp.go`
- `core/abstract/*.go`

### 3. Resource Cleanup
- [ ] Check for goroutine leaks beyond `streamChannel`
- [ ] Verify channel closure patterns
- [ ] Check for unclosed resources (files, connections)
- [ ] Verify context cancellation handling

**Files to examine:**
- `core/interface/http/register.go`
- `core/runtime/dispatch/*.go`
- `core/runtime/notification/*.go`

### 4. Type Assertions
- [ ] Identify unsafe `.(type)` assertions
- [ ] Check for missing comma-ok patterns
- [ ] Verify type assertion safety in dispatch
- [ ] Check HTTP request body type handling

**Files to examine:**
- `core/runtime/dispatch/*.go`
- `core/interface/http/*.go`
- `core/internal/util/*.go`

### 5. Boundary Conditions
- [ ] Check for integer overflow in calculations
- [ ] Verify empty slice/map handling
- [ ] Check for nil map writes
- [ ] Verify array/slice bounds checking

**Files to examine:**
- `core/runtime/ratestore/inmemory.go`
- `core/runtime/audit_buffer.go`
- `core/interface/http/router.go`

### 6. HTTP Middleware
- [ ] Check auth bypass vulnerabilities
- [ ] Verify rate limiter race conditions
- [ ] Check session handling correctness
- [ ] Verify CORS header handling

**Files to examine:**
- `core/interface/http/middleware.go`
- `core/interface/http/auth.go`
- `core/interface/http/transport_fasthttp.go`

### 7. Dispatch Chain
- [ ] Check message validation correctness
- [ ] Verify permission check logic
- [ ] Check input sanitization
- [ ] Verify response envelope correctness

**Files to examine:**
- `core/runtime/dispatch/dispatch.go`
- `core/runtime/dispatch/input.go`
- `core/runtime/dispatch/permission.go`

### 8. Config Loading
- [ ] Check env parsing edge cases
- [ ] Verify default value handling
- [ ] Check config validation logic
- [ ] Verify type conversion safety

**Files to examine:**
- `core/runtime/config.go`
- `core/runtime/env.go`
- `core/runtime/defaults.go`

### 9. Document Operations
- [ ] Check for pool exhaustion scenarios
- [ ] Verify invalid field access handling
- [ ] Check schema mismatch detection
- [ ] Verify document serialization correctness

**Files to examine:**
- `core/interface/http/doc.go`
- `core/interface/http/transport_fasthttp.go`
- `core/internal/testutil/inputdoc.go`

### 10. Boot Sequence
- [ ] Check race conditions in initialization
- [ ] Verify singleton access patterns
- [ ] Check for double-initialization bugs
- [ ] Verify shutdown order correctness

**Files to examine:**
- `core/internal/boot/*.go`
- `core/hestia.go`
- `core/runtime/*.go`

## Priority Order

1. **High Priority** (P0):
   - Race conditions in dispatch chain
   - Resource leaks in HTTP handlers
   - Nil pointer dereferences in hot paths

2. **Medium Priority** (P1):
   - Error handling inconsistencies
   - Type assertion safety
   - Boundary condition bugs

3. **Low Priority** (P2):
   - Config loading edge cases
   - Boot sequence issues
   - Document operation edge cases

## Expected Deliverables

1. Devnotes for each identified issue
2. Prioritized list of fixes
3. Test cases for critical bugs
4. Recommendations for prevention

## Timeline

- **Phase 1** (1-2 days): High priority items
- **Phase 2** (2-3 days): Medium priority items
- **Phase 3** (1-2 days): Low priority items
- **Total**: 4-7 days

## Notes

- Focus on production correctness, not style
- Document each issue with reproduction steps
- Prioritize IoT/HFT deployment concerns
- Consider edge cases under high concurrency
