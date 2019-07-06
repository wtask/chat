package chat

import "time"

// event - base event related to source identity
type event struct {
	origin    identity
	eventTime time.Time
}

type messageEvent struct {
	event
	message string
}

type joinEvent struct {
	event
	outbox chan<- string // close outbox to stop outgoing messaging
}

type partAction int

const (
	_ partAction = iota
	partActionLeft
	partActionTimeout
)

type partEvent struct {
	event
	action partAction
}

func (a partAction) String() string {
	switch {
	case a == partActionLeft:
		return "left"
	case a == partActionTimeout:
		return "timeout"
	default:
		return "unknown part action"
	}
}
