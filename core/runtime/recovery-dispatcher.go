package runtime

import (
	"fmt"
	"runtime"

	"github.com/asaidimu/hestia/core/abstract"
	"go.uber.org/zap"
)

// PanicError wraps a recovered panic value and its stack for
// upstream classification. Upstream handlers can detect it via
// errors.As or errors.Is.
type PanicError struct {
	Recovered any
	Stack     []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Recovered)
}

// Unwrap returns the recovered value if it was an error, allowing
// errors.Is / errors.As to reach the original cause.
func (e *PanicError) Unwrap() error {
	if err, ok := e.Recovered.(error); ok {
		return err
	}
	return nil
}

type RecoveryDispatcher struct {
	next   abstract.Dispatcher
	logger *zap.Logger
}

func NewRecoveryDispatcher(next abstract.Dispatcher, logger *zap.Logger) *RecoveryDispatcher {
	return &RecoveryDispatcher{next: next, logger: logger}
}

func (d *RecoveryDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &RecoveryDispatcher{next: next, logger: d.logger}
}

func (d *RecoveryDispatcher) Send(msg abstract.Message) (res *abstract.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 4096)
			n := runtime.Stack(stack, false)
			d.logger.Error("panic recovered in dispatcher",
				zap.String("message", msg.Name()),
				zap.Any("panic", r),
				zap.ByteString("stack", stack[:n]),
			)
			err = &PanicError{Recovered: r, Stack: stack[:n]}
		}
	}()
	return d.next.Send(msg)
}

var _ abstract.Dispatcher = (*RecoveryDispatcher)(nil)
