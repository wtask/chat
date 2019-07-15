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

	history MessageHistory
	// historyGreets - num of history messages which will be send to just connected client
	historyGreets int

	logger Logger
}

// serverOption - initializes optional Server dependencies.
type serverOption func(s *Server) error

// WithMessageHistory - use specified MessageHistory with chat server.
// Parameter `greets` defines the num of history messages which will be send to just connected client.
func WithMessageHistory(h MessageHistory, greets int) serverOption {
	return func(s *Server) error {
		if greets < 0 {
			return fmt.Errorf("chat.WithMessageHistory: negative greets value (%d)", greets)
		}
		s.history = h
		s.historyGreets = greets
		return nil
	}
}

// WithLogger - use specified Logger with chat server.
func WithLogger(l Logger) serverOption {
	return func(s *Server) error {
		s.logger = l
		return nil
	}
}

// setup - sets up server with specified options.
func setup(s *Server, options ...serverOption) error {
	if s == nil {
		return nil
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(s); err != nil {
			return err
		}
	}
	return nil
}

// NewServer - creates new chat server which ready to serve several network listeners.
func NewServer(buildBroker BrokerBuilder, options ...serverOption) (*Server, error) {
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
	if err := setup(s, options...); err != nil {
		return nil, err
	}
	s.handleEvents()
	return s, nil
}

// Serve - starts to serve specified network listener.
// If you will close listener, you should also shutdown server by yourself.
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
			logError(s.logger, "Failed to accept connection:", err)
			continue
		}

		remote := formatAddress(conn.RemoteAddr())
		logInfo(s.logger, "ACCEPTED", remote, connectionID(conn))

		if err := s.broker.KeepConnection(conn); err != nil {
			logError(s.logger, "DROPPED", remote, err)
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
	s.broker.Broadcast(serverMessage(time.Now().UTC(), "Chat is stopping now... Bye!"))
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
			client := connectionID(event.Conn)
			if client == "" {
				// TODO log invalid event
				continue
			}
			msg := formatMessage(event.OriginTime.UTC(), client, event.Message)
			s.broker.Broadcast(msg)
			historyPush(s.history, msg)
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
			client := connectionID(event.Conn)
			if client == "" {
				// TODO log invalid event
				continue
			}
			body := fmt.Sprintf("Client %s has joined", client)
			msg := serverMessage(event.OriginTime.UTC(), body)
			s.broker.Broadcast(msg)
			for _, msg := range historyTail(s.history, 10) {
				s.broker.SendMessage(event.Conn, msg)
			}
			historyPush(s.history, msg)
			logInfo(s.logger, body)
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
			client := connectionID(event.Conn)
			if client == "" {
				// TODO log invalid event
				continue
			}
			body := fmt.Sprintf("Client %s has %s", client, formatPartAction(event.Action))
			msg := serverMessage(event.OriginTime.UTC(), body)
			s.broker.Broadcast(msg)
			historyPush(s.history, msg)
			logInfo(s.logger, body)
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
