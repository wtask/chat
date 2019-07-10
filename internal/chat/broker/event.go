package broker

import (
	"net"
	"time"
)

// NetEvent - base event related to network connection
type NetEvent struct {
	Conn       net.Conn
	OriginTime time.Time
}

// MessageEvent - event for incoming message from the outside
type MessageEvent struct {
	NetEvent
	Message string
}

// JoinEvent - event to register new connection
type JoinEvent struct {
	NetEvent
}

type PartAction int

const (
	_ PartAction = iota
	PartActionLeft
	PartActionTimeout
)

// PartEvent - event to unregister
type PartEvent struct {
	NetEvent
	Action PartAction
}

func (a PartAction) String() string {
	switch {
	case a == PartActionLeft:
		return "left"
	case a == PartActionTimeout:
		return "timeout"
	default:
		return "unknown part action"
	}
}
