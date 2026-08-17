package feed

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/dispatch"
)

// FeedInput carries no resource id.
type FeedInput struct {
	Topic string `input:"arguments.topic"`
}

// Event is a streamed item.
type Event struct {
	ID string
}

// StreamEvents streams events for a topic.
//
// @hestia.register(
//   name="system:feed:event:stream",
//   intent="stream",
//   rule="authenticated",
// )
func (s *FeedService) StreamEvents(ctx context.Context, msg abstract.Message, items <-chan dispatch.Item[Event]) error {
	_ = items
	return nil
}
