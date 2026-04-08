//go:build !windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func runService(name string, run func(ctx context.Context) error) error {
	return runInteractive(run)
}

func runInteractive(run func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		cancel()
	}()

	return run(ctx)
}
