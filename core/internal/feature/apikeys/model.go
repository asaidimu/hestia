package apikeys

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"golang.org/x/crypto/bcrypt"

	"github.com/asaidimu/hestia/core/identity"
	"github.com/asaidimu/hestia/core/runtime"
)

const (
	keyLength       = 48
	prefixLength    = 10
	hintLength      = 4
	keyChars        = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	apiKeyCollName  = "_api_key_"
)

type GeneratedKey struct {
	FullKey string
	Prefix  string
	Hash    string
	Hint    string
}

type APIKeyModel struct {
	persistence base.Persistence
}

type CreateKeyRequest struct {
	Name        string         `json:"name"`
	Environment string         `json:"environment,omitempty"`
	Operations  []string       `json:"operations,omitempty"`
	Expiry      string         `json:"expiry,omitempty"`
	Limits      map[string]any `json:"limits,omitempty"`
	IP          map[string]any `json:"ip,omitempty"`
}

type UpdateKeyRequest struct {
	Name        *string         `json:"name,omitempty"`
	Operations  []string        `json:"operations,omitempty"`
	Status      *string         `json:"status,omitempty"`
	Expiry      *string         `json:"expiry,omitempty"`
	Limits      map[string]any  `json:"limits,omitempty"`
	IP          map[string]any  `json:"ip,omitempty"`
	Environment *string         `json:"environment,omitempty"`
}

func NewAPIKeyModel(persistence base.Persistence) *APIKeyModel {
	return &APIKeyModel{persistence: persistence}
}

func (m *APIKeyModel) Generate() (*GeneratedKey, error) {
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

func (m *APIKeyModel) Create(ctx context.Context, key *GeneratedKey, userID string, req *CreateKeyRequest) (*data.Document, error) {
	col, err := m.persistence.Collection(ctx, apiKeyCollName)
	if err != nil {
		return nil, fmt.Errorf("access api_key collection: %w", err)
	}

	fields := map[string]any{
		"name":        req.Name,
		"userId":      userID,
		"prefix":      key.Prefix,
		"hash":        key.Hash,
		"operations":  req.Operations,
		"status":      "active",
		"usage":       0,
		"limits":      req.Limits,
		"ip":          req.IP,
		"environment": req.Environment,
	}
	if req.Expiry != "" {
		fields["expiry"] = req.Expiry
	}

	doc := data.MustNewDocument(fields)
	result, err := col.CreateOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return result.Data, nil
}

func (m *APIKeyModel) List(ctx context.Context, userID string) (data.DocumentSet, error) {
	col, err := m.persistence.Collection(ctx, apiKeyCollName)
	if err != nil {
		return nil, fmt.Errorf("access api_key collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("userId").Eq(userID).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	return result.Data, nil
}

func (m *APIKeyModel) Get(ctx context.Context, keyID, userID string) (*data.Document, error) {
	col, err := m.persistence.Collection(ctx, apiKeyCollName)
	if err != nil {
		return nil, fmt.Errorf("access api_key collection: %w", err)
	}

	q := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(keyID).
		Where("userId").Eq(userID).
		Build()

	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("api key not found")
	}
	return result.Data[0], nil
}

func (m *APIKeyModel) Update(ctx context.Context, keyID, userID string, req *UpdateKeyRequest) (*data.Document, error) {
	col, err := m.persistence.Collection(ctx, apiKeyCollName)
	if err != nil {
		return nil, fmt.Errorf("access api_key collection: %w", err)
	}

	setFields := map[string]any{}
	if req.Name != nil {
		setFields["name"] = *req.Name
	}
	if req.Operations != nil {
		setFields["operations"] = req.Operations
	}
	if req.Status != nil {
		setFields["status"] = *req.Status
	}
	if req.Expiry != nil {
		setFields["expiry"] = *req.Expiry
	}
	if req.Limits != nil {
		setFields["limits"] = req.Limits
	}
	if req.IP != nil {
		setFields["ip"] = req.IP
	}
	if req.Environment != nil {
		setFields["environment"] = *req.Environment
	}

	setDoc := data.Patch(setFields).Document(ctx)
	result, err := col.Update(ctx, &base.CollectionUpdate{
		Set:            setDoc,
		Filter:         query.NewQueryBuilder().Where(data.DocumentIDField).Eq(keyID).Where("userId").Eq(userID).Build().Filters,
		ReturnDocument: true,
	})
	if err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("api key not found")
	}
	return result.Data[0], nil
}

func (m *APIKeyModel) Delete(ctx context.Context, keyID, userID string) error {
	col, err := m.persistence.Collection(ctx, apiKeyCollName)
	if err != nil {
		return fmt.Errorf("access api_key collection: %w", err)
	}

	filter := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(keyID).
		Where("userId").Eq(userID).
		Build().Filters

	deleted, err := col.Delete(ctx, filter, false)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (m *APIKeyModel) Rotate(ctx context.Context, keyID, userID string) (*GeneratedKey, *data.Document, error) {
	col, err := m.persistence.Collection(ctx, apiKeyCollName)
	if err != nil {
		return nil, nil, fmt.Errorf("access api_key collection: %w", err)
	}

	q := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(keyID).
		Where("userId").Eq(userID).
		Build()

	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, nil, fmt.Errorf("get api key: %w", err)
	}
	if result.Count == 0 {
		return nil, nil, fmt.Errorf("api key not found")
	}

	key, err := m.Generate()
	if err != nil {
		return nil, nil, err
	}

	setDoc := data.Patch(map[string]any{
		"prefix": key.Prefix,
		"hash":   key.Hash,
	}).Document(ctx)

	updateResult, err := col.Update(ctx, &base.CollectionUpdate{
		Set:            setDoc,
		Filter:         q.Filters,
		ReturnDocument: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("rotate api key: %w", err)
	}
	if updateResult.Count == 0 {
		return nil, nil, fmt.Errorf("api key not found after update")
	}

	return key, updateResult.Data[0], nil
}

func (m *APIKeyModel) ValidateKey(ctx context.Context, keyString string) (*identity.Claims, error) {
	if len(keyString) < prefixLength {
		return nil, fmt.Errorf("invalid api key")
	}

	prefix := keyString[:prefixLength]

	col, err := m.persistence.Collection(ctx, apiKeyCollName)
	if err != nil {
		return nil, fmt.Errorf("access api_key collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("prefix").Eq(prefix).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query api key: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("invalid api key")
	}

	doc := result.Data[0]
	storedHash, err := doc.GetString("hash")
	if err != nil {
		return nil, fmt.Errorf("invalid api key data")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(keyString)); err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	status, _ := doc.GetString("status")
	if status == "revoked" {
		return nil, fmt.Errorf("api key has been revoked")
	}

	if expiryStr, err := doc.GetString("expiry"); err == nil && expiryStr != "" {
		expiry, err := time.Parse(time.RFC3339, expiryStr)
		if err == nil && time.Now().After(expiry) {
			return nil, fmt.Errorf("api key has expired")
		}
	}

	userID, _ := doc.GetString("userId")
	operations, _ := doc.GetStringArray("operations")
	usage, _ := doc.GetInt("usage")

	now := time.Now().Format(time.RFC3339)
	setDoc := data.Patch(map[string]any{
		"last_used": now,
		"usage":     usage + 1,
	}).Document(ctx)
	col.Update(ctx, &base.CollectionUpdate{
		Set:    setDoc,
		Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(doc.ID()).Build().Filters,
	})

	return &identity.Claims{
		UserID:     userID,
		Operations: operations,
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
