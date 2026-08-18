package updates

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/updater"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/policies"
	usersmodel "github.com/asaidimu/hestia/core/system/users/model"
)

// NoInput is the (empty) input for every updates message — all four messages
// take no arguments.
type NoInput struct{}

type StatusView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Version       string `anansi:"version"`
	StagedVersion string `anansi:"staged_version"`
	Prepared      bool   `anansi:"prepared"`
	LastCheck     int64  `anansi:"last_check"`
}

type ChangelogView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Version   string `anansi:"version"`
	AssetName string `anansi:"asset_name"`
	Changelog string `anansi:"changelog"`
}

type CheckView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Checked   bool   `anansi:"checked"`
	Staged    bool   `anansi:"staged"`
	Version   string `anansi:"version"`
	AutoApply bool   `anansi:"auto_apply"`
}

type ApplyView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message string `anansi:"message"`
}

// Dependencies are what the updates service needs beyond the runtime. The
// service is wired manually (not via the service generator) because it is
// registered only when SelfUpdate is configured.
type Dependencies struct {
	Updater   *updater.Updater
	Store     *Store
	Notifier  abstract.Notifier
	Users     *usersmodel.SystemUsers
	Logger    *zap.Logger
	AppURL    string
	HasMailer bool
	AutoApply bool
	Version   string
}

// Registrations returns the updates service message registrations.
func Registrations(deps Dependencies) []abstract.MessageRegistration {
	svc := NewService(deps.Updater, deps.Store, deps.Notifier, deps.Users, deps.Logger, deps.AppURL, deps.HasMailer, deps.AutoApply, deps.Version)
	return []abstract.MessageRegistration{
		{
			Input:       abstract.Input{Schema: dispatch.SchemaFromType[NoInput]()},
			Name:        "system:updates:status:get",
			Description: "Get self-update status (current and staged version)",
			Intent:      abstract.Read,
			Enabled:     true,
			Output:      dispatch.SchemaFromType[StatusView](),
			Handler:     dispatch.HandleDocument[NoInput, *StatusView](svc.Status),
		},
		{
			Input:       abstract.Input{Schema: dispatch.SchemaFromType[NoInput]()},
			Name:        "system:updates:changelog:get",
			Description: "Get the staged update changelog",
			Intent:      abstract.Read,
			Enabled:     true,
			Output:      dispatch.SchemaFromType[ChangelogView](),
			Handler:     dispatch.HandleDocument[NoInput, *ChangelogView](svc.Changelog),
		},
		{
			Input:       abstract.Input{Schema: dispatch.SchemaFromType[NoInput]()},
			Name:        "system:updates:check:create",
			Description: "Check for and stage an update",
			Intent:      abstract.Create,
			Enabled:     true,
			Output:      dispatch.SchemaFromType[CheckView](),
			Handler:     dispatch.HandleDocument[NoInput, *CheckView](svc.Check),
		},
		{
			Input:       abstract.Input{Schema: dispatch.SchemaFromType[NoInput]()},
			Name:        "system:updates:update:apply",
			Description: "Apply the staged update",
			Intent:      abstract.Create,
			Enabled:     true,
			Output:      dispatch.SchemaFromType[ApplyView](),
			Handler:     dispatch.HandleDocument[NoInput, *ApplyView](svc.Apply),
		},
	}
}

// PolicyBindings binds the four updates messages to the administrator rule.
// They are appended conditionally (only when SelfUpdate is configured), never
// through the static allPolicyBindings in gen_features.go.
func PolicyBindings() []policies.Binding {
	return []policies.Binding{
		{Name: "system:updates:status:get", RuleKey: "administrator", Description: "Get self-update status"},
		{Name: "system:updates:changelog:get", RuleKey: "administrator", Description: "Get the staged update changelog"},
		{Name: "system:updates:check:create", RuleKey: "administrator", Description: "Check for and stage an update"},
		{Name: "system:updates:update:apply", RuleKey: "administrator", Description: "Apply the staged update"},
	}
}