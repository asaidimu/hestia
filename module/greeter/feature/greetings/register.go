package greetings

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
	"github.com/asaidimu/hestia/internal/abstract"
)

type Dependencies struct {
	Store *GreetingStore
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "greeter:greetings:salutation:create", Handler: NewCreateSalutationHandler(deps.Store), Description: "Create a greeting salutation", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: salutationCreateInputSchema(), Payload: definition.FieldTypeObject}, Output: salutationOutputSchema()},
		{Name: "greeter:greetings:salutation:get", Handler: NewGetSalutationHandler(deps.Store), Description: "Get a salutation by ID", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: salutationGetInputSchema(), Arguments: map[string]definition.FieldType{"id": definition.FieldTypeString}}, Output: salutationOutputSchema()},
		{Name: "greeter:greetings:salutation:list", Handler: NewListSalutationsHandler(deps.Store), Description: "List all salutations", Enabled: true, Intent: registration.Query, Output: salutationListOutputSchema()},
		{Name: "greeter:greetings:greeting:generate", Handler: NewGenerateGreetingHandler(deps.Store), Description: "Generate a personalized greeting", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: greetingGenerateInputSchema(), Payload: definition.FieldTypeObject}, Output: greetingOutputSchema()},
	}
}
