package boot

import (
	"testing"
)

// TestValidateMessageName_RequiresFourSegments pins the message-name grammar:
// every message must be module:feature:scope:action (4 segments). This keeps
// routes derivable and prevents collisions between features.
func TestValidateMessageName_RequiresFourSegments(t *testing.T) {
	valid := []string{
		"sys:api:key:create",
		"system:blobs:blob:list",
		"system:auth:session:create",
		"feature:scope:action:verb",
	}
	for _, name := range valid {
		if err := validateMessageName(name); err != nil {
			t.Errorf("validateMessageName(%q) unexpected error: %v", name, err)
		}
	}

	invalid := []string{
		"",
		"nodots",
		"a:b",
		"a:b:c",
		"collections:_user:read", // only 3 segments
		"a:b:c:d:e",
		"sys:api:key", // only 3 segments
	}
	for _, name := range invalid {
		if err := validateMessageName(name); err == nil {
			t.Errorf("validateMessageName(%q) should fail (want 4 segments), got nil", name)
		}
	}
}
