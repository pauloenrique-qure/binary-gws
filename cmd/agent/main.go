package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/binary-gws/agent/internal/collector"
	"github.com/binary-gws/agent/internal/config"
	"github.com/binary-gws/agent/internal/logging"
	"github.com/binary-gws/agent/internal/platform"
	"github.com/binary-gws/agent/internal/scheduler"
	"github.com/binary-gws/agent/internal/transport"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	configPath := flag.String("config", "/etc/gw-agent/config.yaml", "Path to configuration file")
	once := flag.Bool("once", false, "Send one heartbeat and exit")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	dryRun := flag.Bool("dry-run", false, "Build payload and print to stdout without sending")
	printVersion := flag.Bool("print-version", false, "Print version information and exit")
	flag.Parse()

	if *printVersion {
		fmt.Printf("Gateway Agent\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger := logging.New(logging.ParseLevel(*logLevel), os.Stdout, cfg.UUID)

	platformInfo := platform.Detect(cfg.Platform.PlatformOverride)
	logger.Info("Starting Gateway Agent", map[string]interface{}{
		"version":   Version,
		"commit":    Commit,
		"platform":  platformInfo.Platform,
		"os":        platformInfo.OS,
		"arch":      platformInfo.Arch,
		"config":    *configPath,
	})

	collector := collector.New(cfg.Intervals.ComputeSeconds)

	transportClient, err := transport.New(transport.Config{
		APIURL:             cfg.APIURL,
		TokenCurrent:       cfg.Auth.TokenCurrent,
		TokenGrace:         cfg.Auth.TokenGrace,
		CABundlePath:       cfg.TLS.CABundlePath,
		InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
	})
	if err != nil {
		logger.Error("Failed to create transport client", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	sched := scheduler.New(scheduler.Config{
		UUID:             cfg.UUID,
		ClientID:         cfg.ClientID,
		SiteID:           cfg.SiteID,
		Platform:         platformInfo,
		HeartbeatSeconds: cfg.Intervals.HeartbeatSeconds,
		Collector:        collector,
		Transport:        transportClient,
		Logger:           logger,
		Version:          Version,
		Commit:           Commit,
		BuildDate:        BuildDate,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if *once || *dryRun {
		if err := sched.SendOnce(ctx, *dryRun); err != nil {
			logger.Error("Failed to send heartbeat", map[string]interface{}{
				"error": err.Error(),
			})
			os.Exit(1)
		}
		logger.Info("Single heartbeat completed", nil)
		os.Exit(0)
	}

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal", nil)
		cancel()
	}()

	if err := sched.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("Scheduler error", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	logger.Info("Gateway Agent stopped", nil)
}
