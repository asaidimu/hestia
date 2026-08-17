package auth

import (
	"testing"
	"time"
)

// BenchmarkSessionService_Create measures session token issuance cost.
func BenchmarkSessionService_Create(b *testing.B) {
	svc := NewSessionService(testSecret)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := svc.Create("u1", time.Hour, 0); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSessionService_Validate measures HMAC verify + decode on every
// authenticated request — the hot path of request authentication.
func BenchmarkSessionService_Validate(b *testing.B) {
	svc := NewSessionService(testSecret)
	token, _, err := svc.Create("u1", time.Hour, 0)
	if err != nil {
		b.Fatalf("Create: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Validate(token); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCredentialsProvider_ValidateSession is the full provider-level
// validation used by the HTTP middleware.
func BenchmarkCredentialsProvider_ValidateSession(b *testing.B) {
	prov := NewCredentialsProvider(NewSessionService(testSecret), "reset-secret")
	token, _, err := prov.CreateSession("u1", time.Hour)
	if err != nil {
		b.Fatalf("CreateSession: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := prov.ValidateSession(token); err != nil {
			b.Fatal(err)
		}
	}
}