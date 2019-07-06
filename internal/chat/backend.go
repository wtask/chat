package chat

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/wtask/chat/internal/chat/message"

	"github.com/wtask/chat/pkg/background"
)

// MessageHistory - the interface for saving and retriving message history.
type MessageHistory interface {
	// Push - save message for history
	Push(msg string)
	// Tail - peek last several messages from history
	Tail(n int) []string
}

// playground - is an abstraction to help join different goroutines by meaning
type playground struct {
	scope  *background.Scope
	cancel func()
}

func newPlayground() *playground {
	s, c := background.NewScope()
	return &playground{s, c}
}

// Backend - is a chat server core.
type Backend struct {
	launched                  bool
	listener                  net.Listener
	writeTimeout, readTimeout time.Duration
	clientIdentifier          networkIdentifier

	// event channel
	message chan messageEvent
	join    chan joinEvent
	part    chan partEvent

	// connection playgrounds
	connAcceptor *playground // ???
	connWriter   *playground // ???
	connReader   *playground // ???
	// event broadcating playgrounds
	producer  *playground
	consumer  *playground
	messenger *playground

	history MessageHistory
}

// NewBackend - builds new backed for given listener.
func NewBackend(listener net.Listener, history MessageHistory) (*Backend, error) {
	if listener == nil {
		return nil, errors.New("chat.NewBackend: net listener is nil")
	}

	if history == nil {
		return nil, errors.New("chat.NewBackend: invalid message history implementation (nil)")
	}

	return &Backend{
		listener:         listener,
		writeTimeout:     30 * time.Second,
		readTimeout:      60 * time.Second,
		clientIdentifier: defaultNetworkIdentifier,

		message: make(chan messageEvent),
		join:    make(chan joinEvent),
		part:    make(chan partEvent),

		connAcceptor: newPlayground(),
		connWriter:   newPlayground(),
		connReader:   newPlayground(),

		producer:  newPlayground(),
		consumer:  newPlayground(),
		messenger: newPlayground(),

		history: history,
	}, nil
}

func (b *Backend) readyToLaunch() bool {
	return b != nil &&
		!b.launched &&
		b.listener != nil &&
		b.writeTimeout >= 0 &&
		b.readTimeout >= 0 &&
		b.clientIdentifier != nil &&
		b.message != nil &&
		b.join != nil &&
		b.part != nil &&
		b.connAcceptor != nil &&
		b.connWriter != nil &&
		b.connReader != nil &&
		b.producer != nil &&
		b.consumer != nil &&
		b.messenger != nil
}

// throwMessage - starts message propagation from author to all clients.
func (b *Backend) throwMessage(cause event, msg string) {
	if b.producer.scope.Context().Err() != nil {
		// expired
		return
	}
	b.producer.scope.Add(1)
	if cause.eventTime.IsZero() {
		cause.eventTime = time.Now().UTC()
	}
	go func(s *background.Scope) {
		defer s.Done()
		select {
		case b.message <- messageEvent{cause, msg}:
		case <-s.Context().Done():
		}
	}(b.producer.scope)
}

func (b *Backend) throwJoin(id identity, outbox chan<- string) {
	if b.producer.scope.Context().Err() != nil {
		// expired
		return
	}
	b.producer.scope.Add(1)
	go func(s *background.Scope) {
		defer s.Done()
		select {
		case b.join <- joinEvent{event{id, time.Now().UTC()}, outbox}:
		case <-s.Context().Done():
		}
	}(b.producer.scope)
}

func (b *Backend) throwPart(id identity, action partAction) {
	if b.producer.scope.Context().Err() != nil {
		// expired
		return
	}
	b.producer.scope.Add(1)
	go func(s *background.Scope) {
		defer s.Done()
		select {
		case b.part <- partEvent{event{id, time.Now().UTC()}, action}:
		case <-s.Context().Done():
		}
	}(b.producer.scope)
}

func (b *Backend) sendMessage(outbox chan<- string, msg string) <-chan struct{} {
	done := make(chan struct{})
	if b.messenger.scope.Context().Err() != nil {
		close(done)
		return done
	}
	b.messenger.scope.Add(1)
	go func(s *background.Scope) {
		defer func() {
			close(done)
			s.Done()
		}()
		select {
		case outbox <- msg:
		case <-s.Context().Done():
		}
	}(b.messenger.scope)
	return done
}

// Launch - launches chat core in background if no error is returned.
func (b *Backend) Launch() (shutdown func(), startup error) {
	if b == nil {
		return nil, errors.New("chat.Backend: unable to launch when nil")
	}
	if b.launched {
		return nil, errors.New("chat.Backend: already launched")
	}
	if !b.readyToLaunch() {
		return nil, errors.New("server.Backend is not initialized")
	}

	go b.consumeEvents()

	// Backend must be sure all connections has time to close
	connHolder := newPlayground()
	b.connAcceptor.scope.Add(1)
	go func() {
		defer b.connAcceptor.scope.Done()
		for {
			// close listener to stop infinite loop.
			conn, err := b.listener.Accept()
			if err != nil {
				select {
				case <-b.connAcceptor.scope.Context().Done():
					return
				default:
					// TODO process error
				}
				continue
			}

			connHolder.scope.Add(1)
			go func(s *background.Scope) {
				defer s.Done()
				b.holdConnection(conn)
				// DO NOT BLOCK execution here!
				// Otherwise goroutine will wait for scope cancellation
				// even when connection was closed at client side.
			}(connHolder.scope)
		}
	}()

	b.launched = true
	return func() {
			if b == nil || !b.launched {
				return
			}

			b.connAcceptor.cancel()
			b.listener.Close()
			b.connReader.cancel()

			b.throwMessage(event{origin: identity("*SYS*")}, "Bye, chat backend is stopping now...")
			time.Sleep(50 * time.Microsecond)

			b.producer.cancel()
			b.consumer.cancel()
			b.messenger.cancel()
			b.connWriter.cancel()

			connHolder.cancel()

			// may to close event channels

			b.launched = false
		},
		nil
}

func (b *Backend) holdConnection(conn net.Conn) {
	defer conn.Close()
	// start up new connection
	clientID := b.clientIdentifier(conn)
	if clientID == "" {
		// TODO log unexpected behaviour
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	outbox := make(chan string)
	go func() {
		defer wg.Done()
		<-b.upholdOutbox(clientID, conn, outbox)
	}()

	b.throwJoin(clientID, outbox)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-b.upholdInbox(clientID, conn)
	}()

	wg.Wait()
}

func (b *Backend) upholdOutbox(clientID identity, conn net.Conn, outbox <-chan string) <-chan struct{} {
	done := make(chan struct{})
	b.connWriter.scope.Add(1)
	go func(s *background.Scope) {
		defer func() {
			close(done)
			s.Done()
		}()

		if s.Context().Err() != nil {
			return
		}

		reader := strings.NewReader("")
		for {
			select {
			case msg := <-outbox:
				reader.Reset(msg)
			case <-s.Context().Done():
				return
			}
			if reader.Len() == 0 {
				continue
			}
			conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
			_, err := reader.WriteTo(conn)
			if err == nil {
				continue
			}
			netErr, ok := err.(net.Error)
			switch {
			case !ok:
				b.throwPart(clientID, partActionLeft)
				return
			case ok && netErr.Timeout():
				b.throwPart(clientID, partActionTimeout)
				return
			case ok && netErr.Temporary():
				// TODO get msg reminder and append to new msg on next iteration
				// Also we need count the tryouts
				b.throwPart(clientID, partActionLeft)
				return
			}
		}
		// TODO add conn writer code
	}(b.connWriter.scope)
	return done
}

func (b *Backend) upholdInbox(clientID identity, conn net.Conn) <-chan struct{} {
	done := make(chan struct{})
	b.connReader.scope.Add(1)
	go func(s *background.Scope) {
		defer func() {
			close(done)
			s.Done()
		}()

		if s.Context().Err() != nil {
			return
		}

		builder := message.Builder{}
		max, partial := 2048, 1500
		buf := make([]byte, max)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			conn.SetReadDeadline(time.Now().Add(b.readTimeout))
			n, err := conn.Read(buf)
			if n > 0 {
				builder.Write(buf[:n])
			}
			select {
			case <-ticker.C:
				if n > 0 {
					b.throwMessage(event{origin: clientID}, builder.Flush())
				}
			default:
				if n >= partial {
					b.throwMessage(event{origin: clientID}, builder.Flush())
				}
			}
			select {
			case <-s.Context().Done():
				return
			default:
				if err == nil {
					continue
				}
				netErr, ok := err.(net.Error)
				switch {
				case !ok:
					// highly likely io.EOF - client left
					b.throwPart(clientID, partActionLeft)
					return
				case ok && netErr.Timeout():
					b.throwPart(clientID, partActionTimeout)
					return
				case ok && netErr.Temporary():
					continue
				}
			}
		}
	}(b.connReader.scope)
	return done
}

func (b *Backend) consumeEvents() {
	mu := &sync.RWMutex{}
	clients := map[identity]joinEvent{}

	message := func(t time.Time, author identity, body string) string {
		return fmt.Sprintf("[%s] [%s] %s\n", t.Format("15:04:05"), string(author), strings.TrimSuffix(body, "\n"))
	}

	b.consumer.scope.Add(1)
	go func(s *background.Scope) {
		// message consumer
		defer s.Done()
		for {
			select {
			case event := <-b.message:
				msg := message(event.eventTime, event.origin, event.message)
				b.history.Push(msg)
				for _, client := range clients {
					outbox := client.outbox
					mu.RLock()
					b.sendMessage(outbox, msg)
					mu.RUnlock()
				}
			case <-s.Context().Done():
				return
			}
		}
	}(b.consumer.scope)

	b.consumer.scope.Add(1)
	go func(s *background.Scope) {
		// join consumer
		defer s.Done()
		for {
			select {
			case join := <-b.join:
				mu.RLock()
				if _, ok := clients[join.origin]; ok {
					// TODO log duplicate join event
					mu.RUnlock()
					continue
				}
				mu.RUnlock()
				mu.Lock()
				clients[join.origin] = join
				mu.Unlock()
				b.throwMessage(
					event{origin: identity("*SYS*")},
					fmt.Sprintf("Client %s has joined", string(join.origin)),
				)
				go func() {
					for _, msg := range b.history.Tail(10) {
						<-b.sendMessage(join.outbox, msg)
					}
				}()
			case <-s.Context().Done():
				return
			}
		}
	}(b.consumer.scope)

	b.consumer.scope.Add(1)
	go func(s *background.Scope) {
		defer s.Done()
		for {
			select {
			case part := <-b.part:
				mu.RLock()
				if _, ok := clients[part.origin]; !ok {
					// delayed part event?
					mu.RUnlock()
					continue
				}
				mu.RUnlock()
				mu.Lock()
				delete(clients, part.origin)
				mu.Unlock()
				b.throwMessage(
					event{origin: identity("*SYS*")},
					fmt.Sprintf("Client %s has %s", part.origin, part.action),
				)
			case <-s.Context().Done():
				return
			}
		}
	}(b.consumer.scope)
}
