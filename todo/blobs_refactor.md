# Blobs Refactor

## Duplicate Input Types

- **Context:** `core/system/blobs/inputs.go` and `core/system/blobs/model/inputs.go` are byte-for-byte identical (aside from package name). Both define `NsInput`, `NsCreateInput`, `BlobKeyInput`, `BlobListInput`, `BlobUpdateInput`, and other types.
- **Impact:** The handler in `handler.go` uses the package-level `NsCreateInput` (from `blobs/inputs.go`), while the model package has its own copy. Changes to one are silently missed in the other.
- **Files:**
  - `core/system/blobs/inputs.go`
  - `core/system/blobs/model/inputs.go`
- **Plan:**
  1. Decide canonical location (likely `model/inputs.go` since that follows the anansi convention).
  2. Update `handler.go` and any other consumers to import from `model`.
  3. Delete the duplicate `blobs/inputs.go`.
