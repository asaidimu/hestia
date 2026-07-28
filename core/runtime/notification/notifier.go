package notification

import (
	"context"
	"fmt"
	"sync"

	"github.com/asaidimu/hestia/core/abstract"
)

type notifier struct {
	mu       sync.RWMutex
	channels map[abstract.ChannelType]abstract.Channel
}

func New() abstract.Notifier {
	return &notifier{
		channels: make(map[abstract.ChannelType]abstract.Channel),
	}
}

func (n *notifier) RegisterChannel(ch abstract.Channel) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.channels[ch.Type()] = ch
}

func (n *notifier) Send(ctx context.Context, notif abstract.Notification) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(notif.Channels) == 0 {
		return nil
	}

	var errs []error
	for _, ct := range notif.Channels {
		ch, ok := n.channels[ct]
		if !ok {
			errs = append(errs, fmt.Errorf("no channel registered for %q", ct))
			continue
		}
		if err := ch.Send(ctx, notif); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ct, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification send errors: %v", errs)
	}
	return nil
}
