package orchestrator

import "context"

type Orchestrator interface {
	Start(bootstrapped bool)
	Restart(bootstrapped bool)
	Shutdown(ctx context.Context) error
}
