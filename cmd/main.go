package main

import (
	"flag"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rapidmidiex/rmxtui"
)

var serverVar string
var debugVar bool

func init() {
	flag.StringVar(&serverVar, "server", "https://rmx.fly.dev", "API Server Host")
	flag.BoolVar(&debugVar, "debug", false, "Debug mode. Write logs to `debug.log` file")

	flag.Parse()
}

func main() {
	if debugVar {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			log.Fatalf("DEBUG mode, but could not write to 'debug.log:\n%s\n'", err)
		}
		defer f.Close()
	}

	rmxtui.Run(serverVar, debugVar)
}
