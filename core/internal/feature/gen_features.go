package feature

import (
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/audit"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/blobs"
	"github.com/asaidimu/hestia/core/internal/feature/collections"
	"github.com/asaidimu/hestia/core/internal/feature/operations"
	"github.com/asaidimu/hestia/core/internal/feature/policies"
	"github.com/asaidimu/hestia/core/internal/feature/users"
)

var allDefaultPolicyBindings = func() []policies.Policy {
	var bindings []policies.Policy
	for _, op := range allKnownOperations {
		ruleName := op.RuleKey
		if ruleName == "" {
			ruleName = "administrator"
		}
		bindings = append(bindings, policies.Policy{
			OperationName: op.Name,
			RuleName:      ruleName,
			Enabled:       true,
		})
	}
	return bindings
}()

var allKnownOperations = func() []policies.Operation {
	var all []policies.Operation
	all = append(all, apikeys.DefaultOperations()...)
	all = append(all, audit.DefaultOperations()...)
	all = append(all, auth.DefaultOperations()...)
	all = append(all, blobs.DefaultOperations()...)
	all = append(all, collections.DefaultOperations()...)
	all = append(all, operations.DefaultOperations()...)
	all = append(all, policies.DefaultOperations()...)
	all = append(all, users.DefaultOperations()...)
	return all
}()

func collectAllKnownOperations() []policies.Operation {
	return allKnownOperations
}

func collectFeatureRegistrations(m *SystemModule, apiKeyAuth *auth.APIKeyAuthenticator) []abstract.MessageRegistration {
	var all []abstract.MessageRegistration
	var allRegs []abstract.MessageRegistration

	apikeysDeps := apikeys.Dependencies{
		APIKeyModel: m.apiKeyModel,
	}
	all = append(all, apikeys.Registrations(apikeysDeps)...)
	auditDeps := audit.Dependencies{
		Persist: m.persist,
	}
	all = append(all, audit.Registrations(auditDeps)...)
	authDeps := auth.Dependencies{
		UserModel:           m.userModel,
		CredentialsProvider: m.credProv,
		APIKeyAuth:          apiKeyAuth,
		AdminUserID:         m.adminUserID,
		SessionTTL:          m.cfg.SessionTTL,
	}
	all = append(all, auth.Registrations(authDeps)...)
	blobsDeps := blobs.Dependencies{
		BlobStore:    m.blobSvc,
		PolicyBridge: m.policyBridge,
		Registry:     m.disp,
	}
	all = append(all, blobs.Registrations(blobsDeps)...)
	collectionsDeps := collections.Dependencies{
		Persist: m.persist,
		Registry: m.disp,
		Logger: m.opts.Logger,
		PolicyBridge: m.policyBridge,
	}
	all = append(all, collections.Registrations(collectionsDeps)...)
	operationsDeps := operations.Dependencies{
		Logger: m.opts.Logger,
		Disp: m.disp,
		Bootstrapped: func() bool { return m.bootstrapped },
		OnBootstrap: func() {
			m.bootstrapped = true
			if m.opts.OnBootstrapped != nil {
				m.opts.OnBootstrapped()
			}
		},
		OnReset: func() {
			if m.opts.OnReset != nil {
				m.opts.OnReset()
			}
		},
		AuditModel: m.auditModel,
		Persist: m.persist,
		Registrations: &allRegs,
		APIPrefix: m.cfg.APIPrefix,
	}
	all = append(all, operations.Registrations(operationsDeps)...)
	policiesDeps := policies.Dependencies{
		PolicyModel: m.policyModel,
		PermManager: m.permMgr,
		LiveRules:   m.liveRules,
	}
	all = append(all, policies.Registrations(policiesDeps)...)
	usersDeps := users.Dependencies{
		UserModel: m.userModel,
		Persist: m.persist,
	}
	all = append(all, users.Registrations(usersDeps)...)

	allRegs = all
	return all
}
