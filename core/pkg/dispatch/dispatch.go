package dispatch

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/registration"
)

type Request struct {
	Name      string              `json:"name"`
	Arguments map[string]string   `json:"arguments"`
	Modifiers map[string][]string `json:"modifiers,omitempty"`
	Payload   any                 `json:"payload,omitempty"`
}

type Response struct {
	Data   map[string]any `json:"data"`
	Status int            `json:"status"`
}

type SimpleDispatcher struct {
	inner runtime.Dispatcher
}

func New(inner runtime.Dispatcher) *SimpleDispatcher {
	return &SimpleDispatcher{inner: inner}
}

func (d *SimpleDispatcher) Dispatch(req Request) (Response, error) {
	doc := data.MustNewDocument(map[string]any{}, context.Background())

	args := make(map[string]any)
	for k, v := range req.Arguments {
		args[k] = v
	}
	doc.Set("arguments", args)

	mods := make(map[string]any)
	for k, vals := range req.Modifiers {
		if len(vals) > 0 {
			mods[k] = vals[0]
		}
	}
	doc.Set("modifiers", mods)

	if req.Payload != nil {
		doc.Set("payload", req.Payload)
	}

	msg := abstract.NewMessage(req.Name, context.Background(), doc)
	result, err := d.inner.Send(msg)
	if err != nil {
		return Response{Data: map[string]any{"data": nil, "metadata": map[string]any{}}, Status: 500}, err
	}

	return build(result), nil
}

func build(result *registration.Result) Response {
	meta := map[string]any{}
	resp := Response{Status: 200}

	if result == nil {
		resp.Data = map[string]any{"data": nil, "metadata": meta}
		return resp
	}

	switch {
	case result.Document != nil:
		sane, _ := result.Document.Sanitize()
		resp.Data = map[string]any{"data": sane.ToMap(), "metadata": meta}

	case result.Documents != nil:
		items := make([]any, 0, len(result.Documents))
		for _, d := range result.Documents {
			sane, _ := d.Sanitize()
			items = append(items, sane.ToMap())
		}
		resp.Data = map[string]any{"data": items, "metadata": meta}

	case result.Page != nil:
		items := make([]any, 0, len(result.Page.Documents))
		for _, d := range result.Page.Documents {
			sane, _ := d.Sanitize()
			items = append(items, sane.ToMap())
		}
		if p := result.Page.Pagination; p != nil {
			meta["page"] = p
		}
		resp.Data = map[string]any{"data": items, "metadata": meta}

	default:
		resp.Data = map[string]any{"data": nil, "metadata": meta}
	}

	return resp
}
