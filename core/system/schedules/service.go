package schedules

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/system/schedules/model"

	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
)

// SchedulesService is the service for the cron-triggered schedule domain.
type SchedulesService struct {
	model *model.SystemScheduledMessagess
	live  *LiveSchedule
	// registrations is the shared, late-filled message catalog — dereferenced
	// at call time because services are constructed before RegisterServices.
	// Nil in unit tests: target-schema validation is skipped then.
	registrations *[]abstract.MessageRegistration
}

func NewSchedulesService(rt abstract.Container) (*SchedulesService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	live := abstract.MustResolve[*LiveSchedule](rt)
	registrations, _ := abstract.Resolve[*[]abstract.MessageRegistration](rt)

	model.DangerouslyResetSystemScheduledMessagessModel()
	m, err := model.InitSystemScheduledMessagessModel(persist, nil)
	if err != nil {
		return nil, err
	}
	return &SchedulesService{model: m, live: live, registrations: registrations}, nil
}

// Create creates a cron-triggered schedule.
//
// @hestia.register(
//   name="system:schedules:schedule:create",
//   intent="create",
//   rule="authenticated",
//   description="Create a cron-triggered schedule",
// )
func (s *SchedulesService) Create(ctx context.Context, msg abstract.Message, input *model.ScheduleCreateInput) (*model.ScheduleCreatedView, error) {
	claims, ok := runtimecontext.ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	if input.Cron == "" {
		return nil, common.NewSystemError("SCHEDULE_CRON_REQUIRED", "cron is required")
	}
	if err := ValidateCronExpr(input.Cron); err != nil {
		return nil, err
	}
	if err := validateScheduleTarget(s.registrations, input.Message, input.Input); err != nil {
		return nil, err
	}

	userID := input.UserID
	if userID == "" {
		userID = claims.UserID
	}

	tenantID := runtimecontext.GetTenantID(ctx)

	doc := data.MustNewDocument(map[string]any{
		"user_id":  userID,
		"message":  input.Message,
		"input":    input.Input,
		"cron":     input.Cron,
		"disabled": input.Disabled,
	})
	if tenantID != "" {
		doc.Set("tenant_id", tenantID)
	}

	saved, err := s.model.CreateSchedule(ctx, doc)
	if err != nil {
		return nil, err
	}

	s.live.Register(ctx, saved)

	return document.New(&model.ScheduleCreatedView{
		ID:      saved.ID(),
		Message: "schedule created",
	}), nil
}

// List lists all schedules for the current tenant.
//
// @hestia.register(
//   name="system:schedules:schedule:list",
//   intent="read",
//   rule="authenticated",
//   description="List all schedules",
// )
func (s *SchedulesService) List(ctx context.Context, msg abstract.Message, input *model.ScheduleListInput) ([]*document.Document, error) {
	tenantID := runtimecontext.GetTenantID(ctx)

	docs, err := s.model.ListSchedulesByTenant(ctx, tenantID, 50, 0)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// Get returns a single schedule by ID.
//
// @hestia.register(
//   name="system:schedules:schedule:get",
//   intent="read",
//   rule="authenticated",
//   description="Get a single schedule by ID",
//   resource_id="id",
// )
func (s *SchedulesService) Get(ctx context.Context, msg abstract.Message, input *model.ScheduleGetInput) (*model.ScheduleDocumentView, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	schedule, err := s.model.GetSchedule(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, fmt.Errorf("schedule not found")
	}
	var view model.ScheduleDocumentView
	if err := schedule.BindTo(&view); err != nil {
		return nil, err
	}
	view.ID = schedule.ID()
	return document.New(&view), nil
}

// Update updates a schedule.
//
// @hestia.register(
//   name="system:schedules:schedule:update",
//   intent="update",
//   rule="authenticated",
//   description="Update a schedule",
//   resource_id="id",
// )
func (s *SchedulesService) Update(ctx context.Context, msg abstract.Message, input *model.ScheduleUpdateInput) (*model.MessageOutput, error) {
	if input.ID == "" {
		return nil, common.NewSystemError("SCHEDULE_ID_REQUIRED", "id is required")
	}

	existing, err := s.model.GetSchedule(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, common.NewSystemError("SCHEDULE_NOT_FOUND", fmt.Sprintf("schedule %q not found", input.ID))
	}

	// Merge provided fields over the stored schedule so persistence and
	// validation always see the complete post-update state. Absent fields
	// leave the stored value unchanged.
	updates := map[string]any{}
	message, _ := existing.GetString("message")
	cronExpr, _ := existing.GetString("cron")
	var inputMap map[string]any
	if raw, err := existing.Get("input"); err == nil {
		if m, ok := raw.(map[string]any); ok {
			inputMap = m
		}
	}

	targetTouched := false
	if input.Message != "" {
		updates["message"] = input.Message
		message = input.Message
		targetTouched = true
	}
	if input.Input != nil {
		updates["input"] = input.Input
		inputMap = input.Input
		targetTouched = true
	}
	if input.Cron != "" {
		if err := ValidateCronExpr(input.Cron); err != nil {
			return nil, err
		}
		updates["cron"] = input.Cron
		cronExpr = input.Cron
		targetTouched = true
	}
	if input.Disabled != nil {
		updates["disabled"] = *input.Disabled
	}

	if len(updates) == 0 {
		return nil, common.NewSystemError("SCHEDULE_NO_FIELDS", "no updatable fields provided")
	}
	// Only re-validate the dispatch target when a target-affecting field
	// changed; disabling a broken schedule must stay possible.
	if targetTouched {
		if err := validateScheduleTarget(s.registrations, message, inputMap); err != nil {
			return nil, err
		}
	}
	if cronExpr == "" {
		return nil, common.NewSystemError("SCHEDULE_CRON_REQUIRED", "cron is required")
	}

	s.live.UnregisterByID(ctx, input.ID)
	if err := s.model.UpdateSchedule(ctx, input.ID, updates); err != nil {
		return nil, err
	}

	saved, err := s.model.GetSchedule(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, common.NewSystemError("SCHEDULE_NOT_FOUND", fmt.Sprintf("schedule %q not found after update", input.ID))
	}

	s.live.Register(ctx, saved)

	return document.New(&model.MessageOutput{Message: "updated"}), nil
}

// Delete deletes a schedule.
//
// @hestia.register(
//   name="system:schedules:schedule:delete",
//   intent="delete",
//   rule="authenticated",
//   description="Delete a schedule",
//   resource_id="id",
// )
func (s *SchedulesService) Delete(ctx context.Context, msg abstract.Message, input *model.ScheduleDeleteInput) (*model.MessageOutput, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	s.live.UnregisterByID(ctx, input.ID)

	if err := s.model.DeleteSchedule(ctx, input.ID); err != nil {
		return nil, err
	}

	return document.New(&model.MessageOutput{Message: "deleted"}), nil
}
