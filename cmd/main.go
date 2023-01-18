package main

import (
	"flag"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rapidmidiex/rmxtui"
)

var serverVar string

func init() {
	flag.StringVar(&serverVar, "server", "https://rmx.fly.dev", "API Server Host")

	flag.Parse()
}

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			log.Fatalf("DEBUG env var set, but could not write to 'debug.log:\n%s\n'", err)
		}
		defer f.Close()
	}

	rmxtui.Run(serverVar)
}
