package chat

import (
	"crypto/sha1"
	"encoding/hex"
	"net"
)

type identity string

type networkIdentifier func(net.Conn) identity

func defaultNetworkIdentifier(c net.Conn) identity {
	if c == nil {
		return ""
	}
	b := sha1.Sum([]byte(c.RemoteAddr().String()))
	return identity(hex.EncodeToString(b[:]))
}
