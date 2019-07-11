package broker

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func randomSleep(min, max time.Duration) {
	if min > max {
		min, max = max, min
	}
	r := rand.Int63n(int64(max - min))
	time.Sleep(time.Duration(r) + min)
}

func TestBroker__StartStop(test *testing.T) {
	inbox := make(chan<- MessageEvent)
	join := make(chan<- JoinEvent)
	part := make(chan<- PartEvent)
	r := 15 * time.Second
	w := 15 * time.Second
	t := 150 * time.Millisecond
	b, err := New(
		WithInbox(inbox),
		WithJoinChan(join),
		WithPartChan(part),
		WithReadTimeout(r),
		WithWriteTimeout(w),
		WithReadTick(t),
	)
	if err != nil {
		test.Error("broker.New, unexpected error", err)
	}
	if b.ctx == nil {
		test.Error("broker.New: unexpected context used")
	}
	if b.cancel == nil {
		test.Error("broker.New: cancel func is nil")
	}
	if b.inbox != inbox {
		test.Error("broker.New: unexpected inbox-channel")
	}
	if b.join != join {
		test.Error("broker.New: unexpected join-channel")
	}
	if b.part != part {
		test.Error("broker.New: unexpected part-channel")
	}
	if b.readTimeout != r {
		test.Error("broker.New: unexpected read timeout", b.readTimeout)
	}
	if b.writeTimeout != w {
		test.Error("broker.New: unexpected write timeout", b.writeTimeout)
	}
	if b.readTick != t {
		test.Error("broker.New: unexpected ticks duration", b.readTick)
	}
	if b.bufSize <= 0 {
		test.Error("broker.New: invalid bufSize", b.bufSize)
	}
	if b.packetSize <= 0 {
		test.Error("broker.New: invalid packetSize", b.packetSize)
	}
	if b.packetSize > b.bufSize {
		test.Error("broker.New: bufSize must be >= packetSize")
	}

	test.Log("Broker stopped in:", b.Quit(5*time.Millisecond))
}

// receiverTest - builds net client to check received messages
func receiverTest(test *testing.T, wg *sync.WaitGroup, expected []string) func(id string, conn net.Conn) {
	return func(id string, conn net.Conn) {
		defer func() {
			conn.Close()
			wg.Done()
			test.Log(id, "done")
		}()
		test.Log(id, "started")
		buf, err := ioutil.ReadAll(conn)
		if err != nil {
			test.Log(id, "connection read error", err)
		}
		received := strings.SplitAfter(string(buf), "\n")
		test.Log(id, "received", len(received), "message(s)", "total", len(buf), "byte(s)")
		if !reflect.DeepEqual(received, expected) {
			test.Error(id, "expected messages:", expected, "received:", received)
		}
	}
}

type link struct{ clientConn, brokerConn net.Conn }

func connect() link {
	c, s := net.Pipe()
	return link{c, s}
}

func TestBroker_KeepConnection_ErrorCase(test *testing.T) {
	link1 := connect()
	cases := []struct {
		link        link
		expectedErr error
	}{
		{link1, nil},
		{link1, ErrConnKept},
	}
	b, err := New()
	if err != nil {
		test.Error("broker.New, unexpected error:", err)
	}
	for _, c := range cases {
		if err := b.KeepConnection(c.link.brokerConn); err != c.expectedErr {
			test.Error("Expected error:", c.expectedErr, "got:", err)
		}
	}
	b.Quit(100 * time.Millisecond)

	for _, l := range []link{link1, connect()} {
		if err := b.KeepConnection(l.brokerConn); err != ErrUnderStopCondition {
			test.Error("Expected error:", ErrUnderStopCondition, "got:", err)
		}
	}
}

func TestBroker_SendMessage(test *testing.T) {
	b, err := New()
	if err != nil {
		test.Error("broker.New, unexpected error:", err)
	}

	clientConn, brokerConn := net.Pipe()
	messages := []string{
		"message-1\n",
		"message 2",
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	client := receiverTest(test, wg, messages)
	go client("net-client", clientConn)

	b.KeepConnection(brokerConn)
	test.Log("broker started")
	b.SendMessage(brokerConn, messages[0])
	test.Logf("broker sent %q", messages[0])
	b.SendMessage(brokerConn, messages[1])
	test.Logf("broker sent %q", messages[1])

	test.Log("broker stopped in:", b.Quit(100*time.Millisecond))

	wg.Wait()
}

func TestBroker_Broadcast(test *testing.T) {
	network := []link{
		connect(),
		connect(),
	}

	b, err := New()
	if err != nil {
		test.Error("broker.New, unexpected error:", err)
	}
	messages := []string{
		"message-1\n",
		"message 2",
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(network))
	client := receiverTest(test, wg, messages)
	for i, l := range network {
		go func(conn net.Conn) {
			client(fmt.Sprintf("net-client-%d", i+1), conn)
		}(l.clientConn)
		b.KeepConnection(l.brokerConn)
		test.Log("broker start to keep connection #", i+1)
	}

	for _, m := range messages {
		b.Broadcast(m)
	}

	test.Log("broker stopped in:", b.Quit(100*time.Millisecond))

	wg.Wait()
}
