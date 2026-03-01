package main

import (
	"fmt"
	"os"

	"github.com/allataetm-svg/goclaw/internal/channel"
	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/manage"
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
	case "gateway":
		gateway()
	case "manage":
		manage.Run()
	case "pairing":
		handlePairingCommand()
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
  gateway - Starts the multi-channel gateway (Telegram, Console, etc.).
  manage  - Opens the interactive agent/channel management dashboard.
  pairing - Manages user authorizations (approve <channel> <userID> <code>).
  help    - Shows this help message.

Example Usage:
  ./goclaw onboard
  ./goclaw gateway
  ./goclaw pairing approve Telegram 123456 000000`)
}

func gateway() {
	conf, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	router := channel.NewRouter(conf)

	// Always add the default Console channel for local use
	console := channel.NewConsoleChannel("cli", "Main Console", conf.DefaultAgent)
	router.RegisterChannel(console)

	// Load other channels from config
	for _, cc := range conf.Channels {
		var ch channel.Channel
		switch cc.Type {
		case "telegram":
			token := cc.Settings["token"]
			if token == "" {
				fmt.Printf("Warning: Skipping channel %s, token not found\n", cc.ID)
				continue
			}
			ch = channel.NewTelegramChannel(cc.ID, cc.Name, token, cc.AgentID)
		default:
			fmt.Printf("Warning: Unknown channel type %s for %s\n", cc.Type, cc.ID)
			continue
		}

		router.RegisterChannel(ch)
		fmt.Printf("Registered channel: %s (%s)\n", cc.Name, cc.Type)
	}

	fmt.Println("🚀 GoClaw Gateway Started. Listening for messages...")
	if err := router.Start(); err != nil {
		fmt.Printf("Gateway error: %v\n", err)
		return
	}

	// Keep alive
	select {}
}

func handlePairingCommand() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: goclaw pairing approve <channel> <userID> <code>")
		return
	}

	sub := os.Args[2]
	if sub != "approve" {
		fmt.Printf("Unknown pairing subcommand: %s\n", sub)
		return
	}

	// channel := os.Args[3] // We don't strictly need channel name for approval but good for UX
	userID := os.Args[4]
	code := os.Args[5]

	if err := config.ApprovePairing(userID, code); err != nil {
		fmt.Printf("❌ Approval failed: %v\n", err)
	} else {
		fmt.Printf("✅ User %s successfully approved and added to whitelist.\n", userID)
	}
}
