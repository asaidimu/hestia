package runtime

import (
	"fmt"
	"runtime"

	"github.com/asaidimu/hestia/core/registration"
	"go.uber.org/zap"
)

type RecoveryDispatcher struct {
	next   Dispatcher
	logger *zap.Logger
}

func NewRecoveryDispatcher(next Dispatcher, logger *zap.Logger) *RecoveryDispatcher {
	return &RecoveryDispatcher{next: next, logger: logger}
}

func (d *RecoveryDispatcher) Send(msg Message) (res *registration.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 4096)
			n := runtime.Stack(stack, false)
			d.logger.Error("panic recovered in dispatcher",
				zap.String("message", msg.Name()),
				zap.Any("panic", r),
				zap.ByteString("stack", stack[:n]),
			)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return d.next.Send(msg)
}

var _ Dispatcher = (*RecoveryDispatcher)(nil)
