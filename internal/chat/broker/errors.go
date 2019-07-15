package broker

import "errors"

var (
	// ErrUnderStopCondition - returns in case if Broker is under stop condition
	// and will not accept any new connections, so you should close such connection by your own.
	ErrUnderStopCondition = errors.New("broker.Broker: under stop condition")

	// ErrConnKept - returns in case if connection is kept already.
	// Do not close such connection after this error, otherwise Broker will drop it.
	ErrConnKept = errors.New("broker.Broker: connection is kept already")
)
