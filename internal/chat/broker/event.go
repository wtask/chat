package broker

import (
	"net"
	"time"
)

// NetEvent - base event related to network connection.
type NetEvent struct {
	Conn       net.Conn
	OriginTime time.Time
}

// MessageEvent - occurres when message from the outside was arrived.
type MessageEvent struct {
	NetEvent
	Message string
}

// JoinEvent - occurres after new client was connected.
type JoinEvent struct {
	NetEvent
}

// PartAction - describes the type of parting with client (connection).
type PartAction int

const (
	_ PartAction = iota
	// PartActionLeft - the parting is occurred due to connection was closed.
	PartActionLeft
	// PartActionTimeout - the parting is occurred due to connection timeout.
	PartActionTimeout
)

// PartEvent - occurres after parting with client.
type PartEvent struct {
	NetEvent
	Action PartAction
}
