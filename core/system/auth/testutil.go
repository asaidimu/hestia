package auth

import (
	"time"

	"github.com/asaidimu/hestia/core/abstract"
	apikeysmodel "github.com/asaidimu/hestia/core/system/apikeys/model"
	usersmodel "github.com/asaidimu/hestia/core/system/users/model"
)

// NewAuthServiceForTest creates an AuthService with explicit dependencies for testing.
func NewAuthServiceForTest(
	users *usersmodel.SystemUsers,
	apiKeys *apikeysmodel.SystemAPIKeys,
	credProv abstract.CredentialsProvider,
	apiKeyAuth *APIKeyAuthenticator,
	adminUserID string,
	sessionTTL time.Duration,
	blocklist *TokenBlocklist,
) *AuthService {
	return &AuthService{
		users:       users,
		apiKeys:     apiKeys,
		credProv:    credProv,
		apiKeyAuth:  apiKeyAuth,
		adminUserID: adminUserID,
		sessionTTL:  sessionTTL,
		blocklist:   blocklist,
	}
}

// TestSetNotifier sets the notifier for testing.
func (s *AuthService) TestSetNotifier(n abstract.Notifier, appURL string) {
	s.notifier = n
	s.appURL = appURL
}

// TestUsers returns the users model for testing.
func (s *AuthService) TestUsers() *usersmodel.SystemUsers { return s.users }

// TestCredProv returns the credentials provider for testing.
func (s *AuthService) TestCredProv() abstract.CredentialsProvider { return s.credProv }
