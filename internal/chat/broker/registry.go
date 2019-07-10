package broker

import (
	"context"
	"net"
	"sync"
)

type client struct {
	ctx    context.Context
	outbox chan string
}

type registry struct {
	mu   sync.RWMutex
	list map[net.Conn]*client
}

func newRegistry() *registry {
	return &registry{
		list: make(map[net.Conn]*client),
	}
}

func (r *registry) len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.list)
}

func (r *registry) get(conn net.Conn) (c *client, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok = r.list[conn]
	return c, ok
}

func (r *registry) add(conn net.Conn, c *client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.list[conn]; ok {
		return false
	}
	r.list[conn] = c
	return true
}

func (r *registry) delete(conn net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.list, conn)
}

func (r *registry) scan(f func(net.Conn)) {
	for conn := range r.list {
		f(conn)
	}
}
