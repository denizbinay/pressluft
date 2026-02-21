package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"pressluft/internal/devserver"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("error=%v", err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: pressluft <dev|serve> [--port <port>]")
	}

	command := args[0]
	switch command {
	case "dev", "serve":
		return runServer(args[1:])
	default:
		return fmt.Errorf("unknown command %q (expected dev or serve)", command)
	}
}

func runServer(args []string) error {
	flags := flag.NewFlagSet("pressluft", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	port := flags.Int("port", 8080, "HTTP port")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	logger := log.New(os.Stdout, "pressluft ", log.LstdFlags|log.LUTC)
	addr := fmt.Sprintf(":%d", *port)
	server := devserver.New(addr, logger)
	return server.Start()
}
