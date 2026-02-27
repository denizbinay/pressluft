package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"pressluft/internal/agent"
)

var (
	configPath = flag.String("config", "/etc/pressluft/agent.yaml", "path to config file")
	register   = flag.String("register", "", "registration token (if not already registered)")
)

func main() {
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := agent.LoadConfig(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if *register != "" {
		cfg.RegistrationToken = *register
		if err := agent.Register(cfg, *configPath); err != nil {
			logger.Error("registration failed", "error", err)
			os.Exit(1)
		}
		logger.Info("registration successful")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutting down")
		cancel()
	}()

	agent := agent.New(cfg, logger)
	if err := agent.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("agent error", "error", err)
		os.Exit(1)
	}
}
