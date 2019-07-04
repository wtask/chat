package background

import (
	"context"
	"sync"
)

// Scope - abstract concurrency scope
type Scope struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	scope     sync.WaitGroup
}

// NewScope - concurrency scope builder
func NewScope() (scope *Scope, cancel func()) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	b := &Scope{
		ctx:       ctx,
		ctxCancel: cancelFunc,
		scope:     sync.WaitGroup{},
	}
	return b,
		func() {
			b.ctxCancel()
			b.scope.Wait()
		}
}

// Context - return background context
func (s *Scope) Context() context.Context {
	return s.ctx
}

// Add - notifies scope to register processes/workers/layers.
// Based on sync.WaitGroup.
func (s *Scope) Add(delta int) {
	s.scope.Add(delta)
}

// Done - notifies scope when process/worker/layer is done.
// Based on sync.WaitGroup.
func (s *Scope) Done() {
	s.scope.Done()
}
