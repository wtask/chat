package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wtask/chat/pkg/semver"
)

type (
	// Configuration - server configuration
	Configuration struct {
		// IPAddress - bind the address
		IPAddress string
		// Port - bind the port
		Port uint
		// ClientIdleTimeout - idle period before client is disconnected
		ClientIdleTimeout time.Duration
		// ClientHistoryGreets - num of messages from chat history which is pushed to newly connected client
		ClientHistoryGreets int
	}
)

const (
	// IdleTimeoutMultiplier - timeout payload without time units
	IdleTimeoutMultiplier = 60
)

var (
	// Config - current configuration of the server
	Config = Configuration{
		IPAddress:           "",
		Port:                20000,
		ClientIdleTimeout:   IdleTimeoutMultiplier * time.Second,
		ClientHistoryGreets: 10,
	}

	// BinaryName - name of run application binary
	BinaryName = strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))

	// Version - app version fingerprint
	Version = semver.V{Minor: 3, Patch: 1, PreRelease: ""}.String()
)

func init() {
	out := flag.CommandLine.Output()
	printUsage := func() {
		fmt.Fprintf(out, "Launch text chat server over TCP\n\n\t%s [options]\nOptions:\n\n", BinaryName)
		flag.PrintDefaults()
		fmt.Fprint(out, "\n")
	}
	printError := func(msg string) {
		fmt.Fprintf(out, "%s (v%s) error:\n\n\t%s\n", BinaryName, Version, msg)
	}

	help := false
	flag.BoolVar(&help, "help", false, "Print usage help")
	flag.StringVar(&Config.IPAddress, "ip", "", "Listen address")
	flag.UintVar(&Config.Port, "port", 20000, "Listen port")
	clientTTL := IdleTimeoutMultiplier
	flag.IntVar(&clientTTL, "client-timeout", clientTTL, "Idle duration in seconds before client is disconnected.")
	flag.IntVar(
		&Config.ClientHistoryGreets,
		"history-greets",
		10,
		"Num of messages from chat history which is pushed to newly connected client.",
	)

	flag.Parse()

	if help {
		printUsage()
		os.Exit(0)
	}

	if clientTTL < 1 {
		printError("client-timeout value should be greater 1")
		os.Exit(1)
	}
	Config.ClientIdleTimeout = time.Duration(clientTTL) * time.Second

	if Config.ClientHistoryGreets < 0 {
		printError("history-greets value should be greater or equal 0")
		os.Exit(1)
	}

	fmt.Fprint(out, "TCP chat server is launching, press Ctrl-C to stop...\n")
}
