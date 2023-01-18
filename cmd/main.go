package main

import (
	"flag"

	"github.com/rapidmidiex/rmxtui"
)

var serverVar string

func init() {
	flag.StringVar(&serverVar, "server", "https://rmx.fly.dev", "API Server Host")

	flag.Parse()
}

func main() {
	rmxtui.Run(serverVar)
}
