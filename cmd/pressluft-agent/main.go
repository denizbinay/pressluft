package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"pressluft/internal/agent"
	"pressluft/internal/envconfig"
	"pressluft/internal/platform"
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
	slog.SetDefault(logger)

	runtimeConfig, err := envconfig.ResolveAgentRuntime(isDevBuild(), *configPath)
	if err != nil {
		logger.Error("failed to resolve agent config", "error", err)
		os.Exit(1)
	}
	executionMode := runtimeConfig.ExecutionMode
	logExecutionMode(logger, executionMode)

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

func logExecutionMode(logger *slog.Logger, mode platform.ExecutionMode) {
	logger.Info("platform contract loaded",
		"execution_mode", mode,
		"contract_package", "pressluft/internal/contract",
	)

	switch mode {
	case platform.ExecutionModeDev:
		logger.Info("development agent transport enabled", "agent_trust", "dev websocket token")
	case platform.ExecutionModeProductionBootstrap:
		logger.Warn("production bootstrap path requires control plane TLS and mTLS trust material", "agent_transport", "wss plus mTLS", "status", "experimental")
	}
}
