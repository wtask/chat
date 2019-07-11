package broker

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/wtask/chat/internal/chat/message"
)

// Broker - chat connections keeper and message router
type Broker struct {
	readTimeout,
	readTick,
	writeTimeout time.Duration
	bufSize, packetSize int
	inbox               chan<- MessageEvent
	join                chan<- JoinEvent
	part                chan<- PartEvent

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	clients *registry
}

type brokerOption func(b *Broker) error

func setup(b *Broker, options ...brokerOption) error {
	if b == nil {
		return nil
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(b); err != nil {
			return err
		}
	}
	return nil
}

// New - builds Broker with needed options.
func New(options ...brokerOption) (*Broker, error) {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Broker{
		readTimeout:  60 * time.Second,
		readTick:     100 * time.Millisecond,
		writeTimeout: 60 * time.Second,
		bufSize:      2048,
		packetSize:   1500,
		ctx:          ctx,
		cancel:       cancel,
		wg:           &sync.WaitGroup{},
		clients:      newRegistry(),
	}

	if err := setup(b, options...); err != nil {
		return nil, err
	}

	return b, nil
}

// Quit - cancels internal context and waits all IO handlers will stop.
// Returns duration of time spent for quit. This time always less or equal of given timeout.
func (b *Broker) Quit(timeout time.Duration) time.Duration {
	if b.ctx.Err() != nil {
		return 0
	}
	from := time.Now()
	b.cancel()
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
	return time.Since(from)
	// TODO close(inbox), close(join), close(part)
}

// KeepConnection - registers new net connection and starts in background IO handlers to communicate over it.
func (b *Broker) KeepConnection(conn net.Conn) error {
	if b.ctx.Err() != nil {
		return ErrUnderStopCondition
	}

	// ctx - new context, derrived from Broker, to help cancel "read" when "write" failed ond vice versa
	ctx, cancelIO := context.WithCancel(b.ctx)
	outbox := make(chan string)
	if !b.clients.add(conn, &client{ctx, outbox}) {
		cancelIO()
		return ErrConnKept
	}

	b.wg.Add(1)
	go func() { // IO handler
		wgIO := sync.WaitGroup{}
		defer func() {
			wgIO.Wait()
			b.clients.delete(conn)
			b.wg.Done()
		}()

		wgIO.Add(1)
		go func() {
			defer func() {
				cancelIO()
				//help to immediately release conn by related reader (inbox) even if its timeout is not expired
				conn.Close()
				wgIO.Done()
			}()
			b.maintainOutbox(ctx, conn, outbox)
		}()

		b.notifyJoin(conn)

		wgIO.Add(1)
		go func() {
			defer func() {
				cancelIO()
				//help to immediately release conn by related writer (writer) even if its timeout is not expired
				conn.Close()
				wgIO.Done()
			}()
			b.maintainInbox(ctx, conn)
		}()
	}()

	return nil
}

// notifyJoin - propagates join event if join channel available.
func (b *Broker) notifyJoin(conn net.Conn) {
	if b.join == nil || b.ctx.Err() != nil {
		return
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		select {
		case b.join <- JoinEvent{NetEvent{conn, time.Now().UTC()}}:
		case <-b.ctx.Done():
		}
	}()
}

// notifyPart - propagates part event if part channel available.
func (b *Broker) notifyPart(conn net.Conn, action PartAction) {
	if b.part == nil || b.ctx.Err() != nil {
		return
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		select {
		case b.part <- PartEvent{NetEvent{conn, time.Now().UTC()}, action}:
		case <-b.ctx.Done():
		}
	}()
}

// notifyInboundMessage - propagates inbound message event through inbox channel.
func (b *Broker) notifyInboundMessage(conn net.Conn, message string) {
	if b.inbox == nil || b.ctx.Err() != nil || message == "" {
		return
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		select {
		case b.inbox <- MessageEvent{NetEvent{conn, time.Now().UTC()}, message}:
		case <-b.ctx.Done():
		}
	}()
}

// SendMessage - tries to send message outwards for single known connection.
func (b *Broker) SendMessage(conn net.Conn, message string) {
	client, ok := b.clients.get(conn)
	if !ok || client.ctx.Err() != nil {
		return
	}
	b.wg.Add(1)
	defer b.wg.Done()
	select {
	case client.outbox <- message:
	case <-client.ctx.Done():
	}
}

// Broadcast - tries to send a message to all kept connections.
// Be careful placing method into goroutine since clients will become to receive unordered messages.
func (b *Broker) Broadcast(message string) {
	wg := sync.WaitGroup{}
	b.clients.scan(func(conn net.Conn) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.SendMessage(conn, message)
		}()
	})
	wg.Wait()
}

func (b *Broker) maintainOutbox(ctx context.Context, conn net.Conn, outbox <-chan string) {
	if ctx.Err() != nil {
		return
	}
	reader := strings.NewReader("")
	for {
		select {
		case message := <-outbox:
			reader.Reset(message)
		case <-ctx.Done():
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
		case ok && netErr.Timeout():
			b.notifyPart(conn, PartActionTimeout)
			return
		default:
			b.notifyPart(conn, PartActionLeft)
			return
		}
	}
}

func (b *Broker) maintainInbox(ctx context.Context, conn net.Conn) {
	if ctx.Err() != nil {
		return
	}

	builder := message.Builder{}
	buf := make([]byte, b.bufSize)
	ticker := time.NewTicker(b.readTick)
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
				b.notifyInboundMessage(conn, builder.Flush())
			}
		default:
			if n >= b.packetSize {
				b.notifyInboundMessage(conn, builder.Flush())
			}
		}
		select {
		case <-ctx.Done():
			return
		default:
			if err == nil {
				continue
			}
			netErr, ok := err.(net.Error)
			switch {
			case ok && netErr.Timeout():
				b.notifyPart(conn, PartActionTimeout)
				return
			default:
				// here when !ok too
				b.notifyPart(conn, PartActionLeft)
				return
			}
		}
	}
}
