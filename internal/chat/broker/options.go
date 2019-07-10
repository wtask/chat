package broker

import (
	"errors"
	"fmt"
	"time"
)

// WithInbox - attach channel to Broker to be notified for incoming messages.
// Note, if Broker is used without inbox it can only to send outgoing messages.
func WithInbox(inbox chan<- MessageEvent) brokerOption {
	return func(b *Broker) error {
		if b.inbox != nil {
			return errors.New("broker.WithInbox: inbox already set up.")
		}
		b.inbox = inbox
		return nil
	}
}

// WithJoinChan - attach channel to Broker to be notified for client joins.
func WithJoinChan(join chan<- JoinEvent) brokerOption {
	return func(b *Broker) error {
		if b.join != nil {
			return errors.New("broker.WithJoinChan: join-channel already set up.")
		}
		b.join = join
		return nil
	}
}

// WithPartChan - attach channel to Broker to be notified for parting with clients.
func WithPartChan(part chan<- PartEvent) brokerOption {
	return func(b *Broker) error {
		if b.part != nil {
			return errors.New("broker.WithPartChan: part-channel already set up.")
		}
		b.part = part
		return nil
	}
}

// WithReadTimeout - overwrite default read timeout of connections.
func WithReadTimeout(timeout time.Duration) brokerOption {
	return func(b *Broker) error {
		if timeout <= 0 {
			return fmt.Errorf("broker.WithReadTimeout: invalid timeout (%v)", timeout)
		}
		b.readTimeout = timeout
		return nil
	}
}

// WithWriteTimeout - overwrite default write timeout of connections.
func WithWriteTimeout(timeout time.Duration) brokerOption {
	return func(b *Broker) error {
		if timeout <= 0 {
			return fmt.Errorf("broker.WithWriteTimeout: invalid timeout (%v)", timeout)
		}
		b.writeTimeout = timeout
		return nil
	}
}

// WithReadTick - overwrite default read tick value of connections.
// Broker checks internal buffer, by default every 100 ms
// and pushes non-empty buffer content into common inbox channel.
func WithReadTick(tick time.Duration) brokerOption {
	return func(b *Broker) error {
		if tick <= 0 {
			return fmt.Errorf("broker.WithReadTicks: invalid tick value (%v)", tick)
		}
		b.readTick = tick
		return nil
	}
}
