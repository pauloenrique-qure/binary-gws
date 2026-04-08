//go:build windows

package main

import (
	"context"
	"os"
	"os/signal"

	"golang.org/x/sys/windows/svc"
)

type gwService struct {
	run func(ctx context.Context) error
}

func (s *gwService) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	done := make(chan error, 1)
	go func() {
		done <- s.run(ctx)
	}()

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
				<-done
				return false, 0
			}
		case <-done:
			return false, 0
		}
	}
}

func runService(name string, run func(ctx context.Context) error) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if isService {
		return svc.Run(name, &gwService{run: run})
	}
	return runInteractive(run)
}

func runInteractive(run func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		cancel()
	}()

	return run(ctx)
}
