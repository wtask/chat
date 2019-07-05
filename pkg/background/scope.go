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
	// TODO Add scope-member counter for stat purposes
}

// NewScope - concurrency scope builder
func NewScope() (scope *Scope, cancel func()) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	s := &Scope{
		ctx:       ctx,
		ctxCancel: cancelFunc,
		scope:     sync.WaitGroup{},
	}
	return s,
		func() {
			if s.Expired() {
				return
			}
			s.ctxCancel()
			s.scope.Wait()
		}
}

// Context - returns background context
func (s *Scope) Context() context.Context {
	return s.ctx
}

// Add - notify the scope to register processes/workers/layers.
// Based on sync.WaitGroup.
func (s *Scope) Add(delta int) {
	s.scope.Add(delta)
}

// Done - notify the scope when process/worker/layer is done.
// Based on sync.WaitGroup.
func (s *Scope) Done() {
	s.scope.Done()
}

// Expired - indicates current state of the scope.
func (s *Scope) Expired() bool {
	select {
	case <-s.Context().Done():
		return true
	default:
		return false
	}
}
