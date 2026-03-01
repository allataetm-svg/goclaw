package main

import (
	"fmt"
	"os"

	"github.com/allataetm-svg/goclaw/internal/channel"
	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/onboard"
	"github.com/allataetm-svg/goclaw/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "onboard":
		onboard.Run()
	case "tui":
		tui.Run()
	case "serve":
		serve()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`🦞 GoClaw - Personal AI Assistant (Lightweight & Local-first)

Available Commands:
  onboard - Starts the setup wizard to configure providers and agents.
  tui     - Starts the Terminal User Interface (Chat).
  serve   - Starts the multi-channel router (Console/Telegram/etc).
  help    - Shows this help message.

Example Usage:
  ./goclaw onboard
  ./goclaw serve`)
}

func serve() {
	conf, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	router := channel.NewRouter(conf)

	// Add a default console channel for this session
	console := channel.NewConsoleChannel("cli", "Main Console", conf.DefaultAgent)
	router.RegisterChannel(console)

	// In a real scenario, we'd load other channels from config here
	// for _, cc := range conf.Channels { ... }

	fmt.Println("Starting GoClaw Multi-Agent Router...")
	if err := router.Start(); err != nil {
		fmt.Printf("Router error: %v\n", err)
		return
	}

	// Wait forever
	select {}
}
