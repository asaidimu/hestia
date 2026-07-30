package runtime

// Metadata keys carried by abstract.Result.Metadata between dispatchers
// and the transport layer. Defined as constants so renaming requires one change.
const (
	MetaKeyRates = "rates"
)

// RateLimitMeta is attached to Result.Metadata[MetaKeyRates] by RateLimitDispatcher.
type RateLimitMeta struct {
	Remaining int   `json:"remaining"`
	Limit     int   `json:"limit"`
	ResetAt   int64 `json:"reset_at"`
}
