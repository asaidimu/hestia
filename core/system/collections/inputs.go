package collections

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
