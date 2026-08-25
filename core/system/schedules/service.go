package schedules

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/system/schedules/model"

	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
)

// SchedulesService is the service for the cron-triggered schedule domain. It
// wraps the hand-rolled ScheduleModel plus the shared LiveSchedule bridge (the
// boot-wired scheduler+dispatcher instance) so create/update/delete keep the
// live cron jobs in sync.
type SchedulesService struct {
	model *model.ScheduleModel
	live  *LiveSchedule
}

func NewSchedulesService(rt abstract.Container) (*SchedulesService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	live := abstract.MustResolve[*LiveSchedule](rt)

	return &SchedulesService{model: model.NewScheduleModel(persist), live: live}, nil
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
		return nil, fmt.Errorf("cron is required")
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

	saved, err := s.model.Create(ctx, doc)
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

	docs, err := s.model.ListByTenant(ctx, tenantID, 50, 0)
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

	schedule, err := s.model.Get(ctx, input.ID)
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
		return nil, fmt.Errorf("id is required")
	}

	s.live.UnregisterByID(ctx, input.ID)
	if err := s.model.Update(ctx, input.ID, map[string]any{
		"message":  input.Message,
		"input":    input.Input,
		"cron":     input.Cron,
		"disabled": input.Disabled,
	}); err != nil {
		return nil, err
	}

	saved, err := s.model.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, fmt.Errorf("schedule not found after update")
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

	if err := s.model.Delete(ctx, input.ID); err != nil {
		return nil, err
	}

	return document.New(&model.MessageOutput{Message: "deleted"}), nil
}