package schedules

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

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

type SchedulesListOutput struct {
	Documents []ScheduleDocumentView `anansi:"documents"`
}

type ScheduleOutput struct {
	Document ScheduleDocumentView `anansi:"document"`
}

type ScheduleCreatedView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string `anansi:"id"`
	Message                string `anansi:"message"`
}

type MessageOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}
