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
	actionLeft
	actionTimeout
)

type partEvent struct {
	event
	action partAction
}

func (a partAction) String() string {
	switch {
	case a == actionLeft:
		return "left"
	case a == actionTimeout:
		return "timeout"
	default:
		return "unknown part action"
	}
}
