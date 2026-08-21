// @note #cruft-20260821-022 issue status=open priority=P2 tags=#cruft,#dead-code : Stale schema functions in schedules/outputs.go
// @see #8uuufn
//
// The schema functions (schedulesListOutputSchema, scheduleOutputSchema,
// messageOutputSchema) are dead code. The generated registrations use
// dispatch.SchemaFromType directly.
//
// Resolution: remove the schema functions. The output types themselves are
// still used by the service methods and registrations.
package schedules

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// ScheduleDocumentView is the declared shape of a schedule document. List and
// get responses carry the raw persisted documents.
type ScheduleDocumentView struct {
	ID        string         `anansi:"_id"`
	UserID    string         `anansi:"user_id"`
	Message   string         `anansi:"message"`
	Input     map[string]any `anansi:"input"`
	Cron      string         `anansi:"cron"`
	Disabled  bool           `anansi:"disabled"`
	TenantID  string         `anansi:"tenant_id"`
	CreatedAt int64          `anansi:"created_at"`
}

// SchedulesListOutput declares the schedules list schema.
type SchedulesListOutput struct {
	Documents []ScheduleDocumentView `anansi:"documents"`
}

func schedulesListOutputSchema() *definition.Schema { return dispatch.SchemaFromType[SchedulesListOutput]() }

// ScheduleOutput declares the single-schedule schema.
type ScheduleOutput struct {
	Document ScheduleDocumentView `anansi:"document"`
}

func scheduleOutputSchema() *definition.Schema { return dispatch.SchemaFromType[ScheduleOutput]() }

// ScheduleCreatedView is the wire shape of a create-schedule response.
type ScheduleCreatedView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string `anansi:"id"`
	Message                string `anansi:"message"`
}

// MessageOutput declares a simple status message response.
type MessageOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}

func messageOutputSchema() *definition.Schema { return dispatch.SchemaFromType[MessageOutput]() }