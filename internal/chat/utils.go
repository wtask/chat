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

func networkID(c net.Conn) string {
	if c == nil {
		return ""
	}
	b := sha1.Sum([]byte(c.RemoteAddr().String()))
	return hex.EncodeToString(b[:])
}

func formatMessage(t time.Time, author, body string) string {
	body = strings.TrimSuffix(body, "\n")
	return fmt.Sprintf("[%s] %s %s\n", t.Format("15:04:05"), author, body)
}

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
