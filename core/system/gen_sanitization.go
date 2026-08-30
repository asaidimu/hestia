package system

import "github.com/asaidimu/go-anansi/v8/core/sanitize"

import (
	apikeysvc "github.com/asaidimu/hestia/core/system/apikeys"
	auditSvc "github.com/asaidimu/hestia/core/system/audit"
	authsvc "github.com/asaidimu/hestia/core/system/auth"
	blobsvc "github.com/asaidimu/hestia/core/system/blobs"
	collectionsvc "github.com/asaidimu/hestia/core/system/collections"
	logssvc "github.com/asaidimu/hestia/core/system/logs"
	notificationsvc "github.com/asaidimu/hestia/core/system/notifications"
	operationsvc "github.com/asaidimu/hestia/core/system/operations"
	policiesvc "github.com/asaidimu/hestia/core/system/policies"
	schedulesvc "github.com/asaidimu/hestia/core/system/schedules"
	settingsvc "github.com/asaidimu/hestia/core/system/settings"
	updatessvc "github.com/asaidimu/hestia/core/system/updates"
	usersvc "github.com/asaidimu/hestia/core/system/users"
	workflowsvc "github.com/asaidimu/hestia/core/system/workflows"
)

// allSanitizationRules aggregates sanitization rules from all features,
// keyed by feature package name. The sanitization dispatcher uses this
// to resolve the scope for a given message.
var allSanitizationRules = func() map[string]*sanitize.FieldMaskConfig {
	m := make(map[string]*sanitize.FieldMaskConfig)
	m["users"] = usersvc.SanitizationRules()
	m["apikeys"] = apikeysvc.SanitizationRules()
	m["audit"] = auditSvc.SanitizationRules()
	m["auth"] = authsvc.SanitizationRules()
	m["logs"] = logssvc.SanitizationRules()
	m["notifications"] = notificationsvc.SanitizationRules()
	m["settings"] = settingsvc.SanitizationRules()
	m["operations"] = operationsvc.SanitizationRules()
	m["policies"] = policiesvc.SanitizationRules()
	m["workflows"] = workflowsvc.SanitizationRules()
	m["schedules"] = schedulesvc.SanitizationRules()
	m["blobs"] = blobsvc.SanitizationRules()
	m["collections"] = collectionsvc.SanitizationRules()
	m["updates"] = updatessvc.SanitizationRules()
	return m
}()
