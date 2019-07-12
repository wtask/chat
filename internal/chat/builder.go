package chat

import (
	"errors"
	"time"

	"github.com/wtask/chat/internal/chat/broker"
)

// BrokerBuilder - helps to build custom broker.Broker with required dependencies (channels)
type BrokerBuilder func(
	inbox chan<- broker.MessageEvent,
	join chan<- broker.JoinEvent,
	part chan<- broker.PartEvent,
) (*broker.Broker, error)

// DefaultBroker - returns builder which requires all event-channels to build broker.Broker.
func DefaultBroker() BrokerBuilder {
	return func(
		inbox chan<- broker.MessageEvent,
		join chan<- broker.JoinEvent,
		part chan<- broker.PartEvent,
	) (*broker.Broker, error) {
		if inbox == nil {
			return nil, errors.New("chat.DefaultBroker: broker.MessageEvent chan is required")
		}
		if join == nil {
			return nil, errors.New("chat.DefaultBroker: broker.JoinEvent chan is required")
		}
		if part == nil {
			return nil, errors.New("chat.DefaultBroker: broker.PartEvent chan is required")
		}
		return broker.New(
			broker.WithInbox(inbox),
			broker.WithJoinChan(join),
			broker.WithPartChan(part),
			broker.WithReadTimeout(60*time.Second),
			broker.WithWriteTimeout(60*time.Second),
			broker.WithReadTick(100*time.Millisecond),
			broker.WithMessageSize(1500, 2048),
		)
	}
}
