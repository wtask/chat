package chat

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/wtask/chat/internal/chat/broker"
)

// Server - represents chat server over any net.Listener implementation.
type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	broker *broker.Broker
	inbox  <-chan broker.MessageEvent
	join   <-chan broker.JoinEvent
	part   <-chan broker.PartEvent
}

// NewServer - creates new chat server which ready to serve several network listeners.
func NewServer(buildBroker BrokerBuilder) (*Server, error) {
	if buildBroker == nil {
		return nil, errors.New("chat.NewServer: required chat.BrokerBuilder is nil")
	}
	inbox := make(chan broker.MessageEvent)
	join := make(chan broker.JoinEvent)
	part := make(chan broker.PartEvent)
	b, err := buildBroker(inbox, join, part)
	if err != nil {
		return nil, fmt.Errorf("chat.NewServer: can't build broker: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
		broker: b,
		inbox:  inbox,
		join:   join,
		part:   part,
	}
	s.handleEvents()
	return s, nil
}

// Serve - starts to serve of specified network listener.
func (s *Server) Serve(listener net.Listener) {
	if listener == nil || s.ctx.Err() != nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.ctx.Done()
		listener.Close()
	}()

	s.wg.Add(1)
	defer s.wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			// TODO process/log error
			continue
		}

		if err := s.broker.KeepConnection(conn); err != nil {
			// == broker.ErrUnderStopCondition
			// TODO panic/log/ignore?
		}
	}
}

// Shutdown - stops server with the specified timeout and returns stopping duration.
// Note, the timeout must consider the duration for stopping the broker and stopping the server itself.
func (s *Server) Shutdown(timeout time.Duration) time.Duration {
	if s.ctx.Err() != nil {
		return 0
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	from := time.Now()
	s.broker.Quit(timeout)
	s.cancel()
	select {
	case <-done:
	case <-time.After(timeout):
	}
	return time.Since(from)
}

func (s *Server) handleMessageEvents() {
	if s.inbox == nil {
		return
	}
	for {
		select {
		case event, ok := <-s.inbox:
			if !ok {
				return
			}
			client := networkID(event.Conn)
			if client == "" {
				// TODO log invalid event
				continue
			}
			s.broker.Broadcast(formatMessage(event.OriginTime.UTC(), client, event.Message))
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Server) handleJoinEvents() {
	if s.join == nil {
		return
	}
	for {
		select {
		case event, ok := <-s.join:
			if !ok {
				return
			}
			client := networkID(event.Conn)
			if client == "" {
				// TODO log invalid event
				continue
			}
			s.broker.Broadcast(
				formatMessage(
					event.OriginTime.UTC(),
					"**SERVER**",
					fmt.Sprintf("Client %s has joined", client),
				),
			)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Server) handlePartEvents() {
	if s.part == nil {
		return
	}
	for {
		select {
		case event, ok := <-s.part:
			if !ok {
				return
			}
			client := networkID(event.Conn)
			if client == "" {
				// TODO log invalid event
				continue
			}
			s.broker.Broadcast(
				formatMessage(
					event.OriginTime.UTC(),
					"**SERVER**",
					fmt.Sprintf("Client %s has %s", client, formatPartAction(event.Action)),
				),
			)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Server) handleEvents() {
	if s.ctx.Err() != nil {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.handleMessageEvents()
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.handleJoinEvents()
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.handlePartEvents()
	}()
}
