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

type (
	// Logger - logging interface used by the server
	Logger interface {
		Println(v ...interface{})
		Printf(format string, v ...interface{})
	}
)

func main() {
	var log Logger = stdlog.New(os.Stdout, "chatsrv:"+Version+" ", stdlog.Ldate|stdlog.Ltime)
	log.Printf("Started with config: %+v", Config)

	node := net.JoinHostPort(Config.IPAddress, fmt.Sprintf("%d", Config.Port))
	listener, err := net.Listen("tcp", node)
	if err != nil {
		log.Println("ERR", "Unable to listen TCP:", err)
		os.Exit(1)
	}
	log.Println("Listen on:", node)

	history, err := history.NewStack(Config.NewClientHistoryGreets)
	if err != nil {
		log.Println("ERR", "Invalid config:", err)
		os.Exit(1)
	}

	srv, err := chat.NewServer(chat.DefaultBroker(), history)
	if err != nil {
		log.Println("ERR", "Can't start chat server:", err)
		listener.Close()
		os.Exit(1)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	go srv.Serve(listener)
	log.Println("Chat server has started.", "Press Ctrl-C in CLI to stop it.")

	<-sig
	log.Println("Got stop signal")
	log.Println("Server stopped in", srv.Shutdown(5*time.Second), "seconds, bye")
}
