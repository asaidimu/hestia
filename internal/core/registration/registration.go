package registration

import "github.com/asaidimu/hestia/internal/abstract"

type Verb = abstract.Verb
type Page = abstract.Page
type Blob = abstract.Blob
type Result = abstract.Result

const (
	Create Verb = abstract.Create
	Read        = abstract.Read
	Update      = abstract.Update
	Delete      = abstract.Delete
	Query       = abstract.Query
	Stream      = abstract.Stream
)
