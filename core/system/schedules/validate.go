package schedules

import (
	"fmt"
	"strings"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	cron "github.com/netresearch/go-cron"

	"github.com/asaidimu/hestia/core/abstract"
)

// ValidateCronExpr checks that a cron expression parses with the same parser
// the runtime scheduler uses (standard syntax plus @every-style descriptors).
func ValidateCronExpr(expr string) error {
	if _, err := cron.ParseStandard(expr); err != nil {
		return common.NewSystemError("SCHEDULE_INVALID_CRON", fmt.Sprintf("invalid cron expression %q: %v", expr, err))
	}
	return nil
}

var (
	validatorMu sync.Mutex
	validators  = map[*definition.Schema]*definition.DocumentValidator{}
)

// strictValidator returns a cached strict validator for an input schema.
func strictValidator(s *definition.Schema) (*definition.DocumentValidator, error) {
	validatorMu.Lock()
	defer validatorMu.Unlock()
	if v, ok := validators[s]; ok {
		return v, nil
	}
	v, err := definition.NewDocumentValidator(s, definition.PredicateMap{})
	if err != nil {
		return nil, err
	}
	validators[s] = v
	return v, nil
}

// validateScheduleTarget checks that message is registered and that input
// satisfies the target handler's declared payload schema. Schedules fire
// unattended, so validation is strict — missing required fields would fail on
// every tick otherwise. A nil registration catalog skips schema validation.
func validateScheduleTarget(regs *[]abstract.MessageRegistration, message string, input map[string]any) error {
	if regs == nil {
		return nil
	}

	var reg *abstract.MessageRegistration
	for i := range *regs {
		if (*regs)[i].Name == message {
			reg = &(*regs)[i]
			break
		}
	}
	if reg == nil {
		return common.NewSystemError("SCHEDULE_UNKNOWN_MESSAGE", fmt.Sprintf("message %q is not registered", message))
	}
	if reg.Input.Schema == nil || len(input) == 0 {
		return nil
	}

	// Plain map, not data.NewDocument — the document factory injects system
	// fields (_id_, _metadata_) that strict schema validation would reject.
	v, err := strictValidator(reg.Input.Schema)
	if err != nil {
		return common.SystemErrorFrom(err).WithOperation("validateScheduleTarget").WithMessage("failed to build validator for message input schema")
	}

	if issues, ok := v.Validate(map[string]any{"payload": input}); !ok {
		msgs := make([]string, 0, len(issues))
		for _, issue := range issues {
			msgs = append(msgs, issue.Message)
		}
		return common.NewSystemError("SCHEDULE_INVALID_INPUT", fmt.Sprintf(
			"input for message %q does not match its schema: %s", message, strings.Join(msgs, "; ")))
	}
	return nil
}
