package model

type CollectionGetInput struct {
	Name string `input:"arguments.name"`
}

type CollectionCreateInput struct {
	Payload map[string]any `input:"payload"`
}

type CollectionDeleteInput struct {
	Name string `input:"arguments.name"`
}

type CollectionDocQueryInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

type CollectionDocCreateInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

type CollectionDocGetInput struct {
	Name  string `input:"arguments.name"`
	DocID string `input:"arguments.doc_id"`
}

type CollectionDocUpdateInput struct {
	Name    string         `input:"arguments.name"`
	DocID   string         `input:"arguments.doc_id"`
	Payload map[string]any `input:"payload"`
}

type CollectionDocDeleteInput struct {
	Name  string `input:"arguments.name"`
	DocID string `input:"arguments.doc_id"`
}

// CollectionDocUpdateManyInput binds a filter-targeted update. The payload
// is a { set, filter } envelope; no doc_id is accepted (which is what keeps
// the derived route free of a /{doc_id} segment).
type CollectionDocUpdateManyInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

// CollectionViewCreateInput binds a view:create request. Payload carries the
// stored query definition ("query") and the view kind ("materialized").
type CollectionViewCreateInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

// CollectionViewRefreshInput binds a view:refresh request.
type CollectionViewRefreshInput struct {
	Name string `input:"arguments.name"`
}