package main

import (
	"fmt"
	stdlog "log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/wtask/chat/internal/chat"
	"github.com/wtask/chat/internal/chat/history"
)

func main() {
	logger := stdlog.New(os.Stdout, "chatsrv:"+Version+" ", stdlog.Ldate|stdlog.Ltime)
	logger.Printf("Started with config: %+v", Config)

	node := net.JoinHostPort(Config.IPAddress, fmt.Sprintf("%d", Config.Port))
	listener, err := net.Listen("tcp", node)
	if err != nil {
		logger.Println("ERR", "Unable to listen TCP:", err)
		os.Exit(1)
	}
	logger.Println("Listen", node)

	history, err := history.NewStack(Config.NewClientHistoryGreets)
	if err != nil {
		logger.Println("ERR", "Invalid config:", err)
		os.Exit(1)
	}

	server, err := chat.NewServer(
		chat.DefaultBroker(),
		chat.WithMessageHistory(history, 10),
		chat.WithLogger(logger),
	)
	if err != nil {
		logger.Println("ERR", "Can't start chat server:", err)
		listener.Close()
		os.Exit(1)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	go server.Serve(listener)
	logger.Println("Chat server has started.")

	<-sig
	logger.Println("Got stop signal")
	logger.Println("Chat server stopped in", server.Shutdown(10*time.Second), "seconds, bye")
}
