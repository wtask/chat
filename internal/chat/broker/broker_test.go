package broker

import (
	"context"
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

func clientMock(ctx context.Context, conn net.Conn, out, in []string, sleepOut func()) {
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

func TestBroker_SendMessage(test *testing.T) {
	b, err := New()
	if err != nil {
		test.Error("broker.New, unexpected error:", err)
	}

	sent := []string{
		"message-1\n",
		"message 2",
	}
	client := func(wg *sync.WaitGroup, conn net.Conn) {
		defer func() {
			conn.Close()
			wg.Done()
		}()
		test.Log("client started")
		buf, err := ioutil.ReadAll(conn)
		if err != nil {
			test.Log("client got read error", err)
		}
		test.Log("client done, read total", len(buf), "byte(s)")
		actual := strings.SplitAfter(string(buf), "\n")
		if !reflect.DeepEqual(actual, sent) {
			test.Error("sent messages:", sent, "client received:", actual)
		}
	}

	clientConn, brokerConn := net.Pipe()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go client(wg, clientConn)

	b.KeepConnection(brokerConn)
	test.Log("broker started")
	b.SendMessage(brokerConn, sent[0])
	test.Logf("broker sent %q", sent[0])
	b.SendMessage(brokerConn, sent[1])
	test.Logf("broker sent %q", sent[1])

	test.Log("broker stopped in:", b.Quit(100*time.Millisecond))

	wg.Wait()
}
