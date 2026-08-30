package audit

import "github.com/asaidimu/go-anansi/v8/core/sanitize"

func SanitizationRules() *sanitize.FieldMaskConfig {
	return &sanitize.FieldMaskConfig{
		DefaultPolicy: sanitize.MaskPreserve,
		Fields: map[string]sanitize.MaskedFieldPolicy{
			"integrity_hash": sanitize.MaskRedact,
		},
	}
}
