package main

import (
	"fmt"
	"os"

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
  help    - Shows this help message.

Example Usage:
  ./goclaw onboard`)
}
