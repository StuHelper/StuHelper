package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/StuHelper/StuHelper/server/internal/campusconnector/node"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "campus connector stopped: operation failed")
		os.Exit(1)
	}
}

func run() error {
	debug.SetTraceback("none")
	_ = syscall.Umask(0o077)
	if err := disableCoreDumps(); err != nil {
		return fmt.Errorf("disable core dumps: %w", err)
	}
	defaultConfig := strings.TrimSpace(os.Getenv("CAMPUS_CONNECTOR_NODE_CONFIG_FILE"))
	configPath := flag.String("config", defaultConfig, "path to the non-secret connector operation configuration")
	flag.Parse()
	cfg, err := node.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	if version != "" && version != "dev" {
		cfg.SoftwareVersion = version
	}
	client, err := node.NewClient(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	runner, err := node.NewRunner(cfg, client, logger)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	logger.Info("campus connector starting",
		"node", cfg.NodeID,
		"software_version", cfg.SoftwareVersion,
		"operation_count", len(cfg.Operations),
	)
	if err := runner.Run(ctx); err != nil {
		return err
	}
	logger.Info("campus connector stopped")
	return nil
}

func disableCoreDumps() error {
	limit := &syscall.Rlimit{Cur: 0, Max: 0}
	return syscall.Setrlimit(syscall.RLIMIT_CORE, limit)
}
