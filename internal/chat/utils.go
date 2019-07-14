package chat

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/wtask/chat/internal/chat/broker"
)

// connectionID - generates network connection identifier.
func connectionID(c net.Conn) string {
	if c == nil {
		return ""
	}
	b := sha1.Sum([]byte(c.RemoteAddr().String()))
	return hex.EncodeToString(b[:])
}

// formatMessage - formats chat message.
func formatMessage(t time.Time, author, body string) string {
	body = strings.TrimSuffix(body, "\n")
	return fmt.Sprintf("[%s] %s %s\n", t.Format("15:04:05"), author, body)
}

// serverMessage - format chat message generated on server.
func serverMessage(t time.Time, body string) string {
	return formatMessage(t, "**SERVER**", body)
}

// formatPartAction - returns string representation of broker.PartAction,
func formatPartAction(a broker.PartAction) string {
	switch a {
	case broker.PartActionTimeout:
		return "timed out"
	case broker.PartActionLeft:
		fallthrough
	default:
		return "left"
	}
}

// formatAddress - formats specified network address for logging purposes.
func formatAddress(a net.Addr) string {
	return fmt.Sprintf("%s %s", a.Network(), a.String())
}
