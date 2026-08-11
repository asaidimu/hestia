package model

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"golang.org/x/crypto/bcrypt"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

const (
	keyLength    = 48
	prefixLength = 10
	hintLength   = 4
	keyChars     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// GeneratedKey carries the raw key material plus the derivable store fields.
type GeneratedKey struct {
	FullKey string
	Prefix  string
	Hash    string
	Hint    string
}

// Generate creates a new random API key and its bcrypt hash. The raw key is
// returned only to the caller and is never persisted.
func (m *SystemAPIKeys) Generate() (*GeneratedKey, error) {
	key, err := randomString(keyLength)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(key), runtime.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash key: %w", err)
	}

	return &GeneratedKey{
		FullKey: key,
		Prefix:  key[:prefixLength],
		Hash:    string(hash),
		Hint:    key[len(key)-hintLength:],
	}, nil
}

// CreateKey persists a new API key from the APIKeyCreate projection, stamping
// the derived key material, owner, and default status.
func (m *SystemAPIKeys) CreateKey(ctx context.Context, key *GeneratedKey, userID string, req *APIKeyCreate) (*SystemAPIKey, error) {
	active := "active"
	usage := int64(0)

	doc := document.New(&SystemAPIKey{
		Name:        req.Name,
		UserID:      userID,
		Prefix:      key.Prefix,
		Hash:        key.Hash,
		Operations:  req.Operations,
		Status:      &active,
		Usage:       &usage,
		Limits:      req.Limits,
		Ip:          req.Ip,
		Environment: req.Environment,
	})
	if req.Expiry != nil {
		doc.Expiry = req.Expiry
	}

	created, err := m.ModelCollection.Create(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return created, nil
}

// List returns all API keys owned by the given user.
func (m *SystemAPIKeys) List(ctx context.Context, userID string) ([]*SystemAPIKey, error) {
	q := query.NewQueryBuilder().Where("userId").Eq(userID).Build()
	keys, err := m.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	return keys, nil
}

// Get returns an API key owned by the given user.
func (m *SystemAPIKeys) Get(ctx context.Context, keyID, userID string) (*SystemAPIKey, error) {
	q := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(keyID).
		Where("userId").Eq(userID).
		Limit(1).
		Build()

	keys, err := m.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("api key not found")
	}
	return keys[0], nil
}

// UpdateKey applies the APIKeyUpdate projection to an owned key, returning the
// updated key.
func (m *SystemAPIKeys) UpdateKey(ctx context.Context, keyID, userID string, req *APIKeyUpdate) (*SystemAPIKey, error) {
	if _, err := m.Get(ctx, keyID, userID); err != nil {
		return nil, err
	}

	updated, err := m.UpdateFrom[*APIKeyUpdate, *SystemAPIKey](ctx, keyID, req)
	if err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}
	return updated, nil
}

// Delete removes an API key owned by the given user.
func (m *SystemAPIKeys) Delete(ctx context.Context, keyID, userID string) error {
	if _, err := m.Get(ctx, keyID, userID); err != nil {
		return err
	}
	return m.DeleteByID(ctx, keyID)
}

// Rotate replaces the key material of an owned key with a freshly generated
// hash, returning the new raw key and the updated key document.
func (m *SystemAPIKeys) Rotate(ctx context.Context, keyID, userID string) (*GeneratedKey, *SystemAPIKey, error) {
	if _, err := m.Get(ctx, keyID, userID); err != nil {
		return nil, nil, err
	}

	key, err := m.Generate()
	if err != nil {
		return nil, nil, err
	}

	updated, err := m.Update(ctx, keyID, &SystemAPIKey{Prefix: key.Prefix, Hash: key.Hash})
	if err != nil {
		return nil, nil, fmt.Errorf("rotate api key: %w", err)
	}
	return key, updated, nil
}

// ValidateKey resolves a raw API key to the claims of its owner, bumping the
// usage counters on every successful validation.
func (m *SystemAPIKeys) ValidateKey(ctx context.Context, keyString string) (*abstract.Claims, error) {
	if len(keyString) < prefixLength {
		return nil, fmt.Errorf("invalid api key")
	}

	prefix := keyString[:prefixLength]

	q := query.NewQueryBuilder().Where("prefix").Eq(prefix).Limit(1).Build()
	keys, err := m.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query api key: %w", err)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("invalid api key")
	}

	k := keys[0]
	if err := bcrypt.CompareHashAndPassword([]byte(k.Hash), []byte(keyString)); err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	if k.Status != nil && *k.Status == "revoked" {
		return nil, fmt.Errorf("api key has been revoked")
	}

	if k.Expiry != nil && *k.Expiry != "" {
		expiry, err := time.Parse(time.RFC3339, *k.Expiry)
		if err == nil && time.Now().After(expiry) {
			return nil, fmt.Errorf("api key has expired")
		}
	}

	usage := int64(0)
	if k.Usage != nil {
		usage = *k.Usage
	}
	now := time.Now().Format(time.RFC3339)
	next := usage + 1
	// Usage bookkeeping is best-effort: a failed bump must not reject a valid key.
	_, _ = m.Update(ctx, k.ID, &SystemAPIKey{LastUsed: &now, Usage: &next})

	return &abstract.Claims{
		UserID:     k.UserID,
		Operations: k.Operations,
		TokenType:  "api_key",
	}, nil
}

func randomString(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(keyChars))))
		if err != nil {
			return "", err
		}
		result[i] = keyChars[n.Int64()]
	}
	return string(result), nil
}
