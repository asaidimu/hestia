package module

import "context"

type BaseModule struct{}

func (BaseModule) Dependencies() []string        { return nil }
func (BaseModule) Start(_ context.Context) error { return nil }
func (BaseModule) Stop(_ context.Context) error  { return nil }
func (BaseModule) Health(_ context.Context) any  { return nil }
