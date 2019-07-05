package chat

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/wtask/chat/pkg/background"
)

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

	// Playgrounds
	// super - helps to resolve complex cases with different related playgrounds
	super *playground
	// connection playgrounds
	connAcceptor *playground
	connWriter   *playground
	connReader   *playground
	// event broadcating playgrounds
	producer  *playground
	consumer  *playground
	messenger *playground
}

// NewBackend - builds new backed for given listener.
func NewBackend(listener net.Listener) (*Backend, error) {
	if listener == nil {
		return nil, errors.New("chat.NewBackend: net listener is nil")
	}

	return &Backend{
		listener:         listener,
		writeTimeout:     30 * time.Second,
		readTimeout:      60 * time.Second,
		clientIdentifier: defaultNetworkIdentifier,

		message: make(chan messageEvent),
		join:    make(chan joinEvent),
		part:    make(chan partEvent),

		super: newPlayground(),

		connAcceptor: newPlayground(),
		connWriter:   newPlayground(),
		connReader:   newPlayground(),

		producer:  newPlayground(),
		consumer:  newPlayground(),
		messenger: newPlayground(),
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
		b.super != nil &&
		b.connAcceptor != nil &&
		b.connWriter != nil &&
		b.connReader != nil &&
		b.producer != nil &&
		b.consumer != nil &&
		b.messenger != nil
}

// propagateMessage - starts message propagation from author to all clients.
func (b *Backend) propagateMessage(cause event, msg string) {
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

func (b *Backend) notifyJoin(id identity, outbox chan<- string) {
	b.producer.scope.Add(1)
	go func(s *background.Scope) {
		defer s.Done()
		select {
		case b.join <- joinEvent{event{id, time.Now().UTC()}, outbox}:
		case <-s.Context().Done():
		}
	}(b.producer.scope)
}

func (b *Backend) notifyPart(id identity, action partAction) {
	b.producer.scope.Add(1)
	go func(s *background.Scope) {
		defer s.Done()
		select {
		case b.part <- partEvent{event{id, time.Now().UTC()}, action}:
		case <-s.Context().Done():
		}
	}(b.producer.scope)
}

func (b *Backend) sendMessage(outbox chan<- string, msg string) {
	// NOTE:
	// We need done-flag to help send consistently if method is calling repeatedly for the same outbox.
	done := make(chan struct{})
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
	<-done
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

	go b.eventDispatcher()

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

			// Wrap connection holder into super scope due to Backend must be sure
			// that the connection has time to close
			b.super.scope.Add(1)
			go func(s *background.Scope) {
				defer s.Done()
				b.holdConnection(conn)
				// do not block execution here
			}(b.super.scope)
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

			b.propagateMessage(event{origin: identity("*SYS*")}, "Bye, chat backend is stopping now...")
			time.Sleep(50 * time.Microsecond)

			b.producer.cancel()
			b.consumer.cancel()
			b.messenger.cancel()
			b.connWriter.cancel()

			b.super.cancel()

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
		b.launchOutbox(clientID, conn, outbox)
	}()

	b.notifyJoin(clientID, outbox)

	wg.Add(1)
	go func() {
		defer wg.Done()
		b.launchInbox(clientID, conn)
	}()

	wg.Wait()
}

func (b *Backend) launchOutbox(clientID identity, conn net.Conn, outbox <-chan string) {
	b.connWriter.scope.Add(1)
	go func(s *background.Scope) {
		defer func() {
			s.Done()
			// close(outbox) // or forget chan
		}()

		// TODO add conn writer code
	}(b.connWriter.scope)
}

func (b *Backend) launchInbox(clientID identity, conn net.Conn) {
	b.connReader.scope.Add(1)
	go func(s *background.Scope) {
		defer s.Done()
		// TODO add conn reader code
	}(b.connReader.scope)
}

func (b *Backend) eventDispatcher() {
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
				// TODO add event to HISTORY
				for _, client := range clients {
					outbox := client.outbox
					mu.RLock()
					go b.sendMessage(outbox, message(event.eventTime, event.origin, event.message))
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
				b.propagateMessage(
					event{origin: identity("*SYS*")},
					fmt.Sprintf("Client %s has joined", string(join.origin)),
				)
				// TODO get HISTORY and send directly to client
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
				b.propagateMessage(
					event{origin: identity("*SYS*")},
					fmt.Sprintf("Client %s has %s", part.origin, part.action),
				)
			case <-s.Context().Done():
				return
			}
		}
	}(b.consumer.scope)
}
