package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/coverage"
	"sync"
	"syscall"

	"github.com/go-logr/logr"
)

var setupCoverageOnce sync.Once

func SetupCoverage(ctx context.Context, logger logr.Logger, signals ...os.Signal) {
	setupCoverageOnce.Do(func() {
		setupCoverage(ctx, logger, signals...)
	})
}

func setupCoverage(ctx context.Context, logger logr.Logger, signals ...os.Signal) {
	dir := os.Getenv("GOCOVERDIR")
	if dir == "" {
		return
	}

	logger = logger.WithValues(slog.String("dir", dir))

	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGUSR1}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)

	logger.Info("coverage enabled")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				err := writeCoverage(dir)
				switch {
				case err != nil:
					logger.Error(err, "write coverage")
				default:
					logger.Info("write coverage")
				}
			}
		}
	}()
}

func writeCoverage(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return fmt.Errorf("glob coverage dir: %w", err)
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("remove coverage file: %w", err)
		}
	}

	if err := coverage.WriteMetaDir(dir); err != nil {
		return fmt.Errorf("write coverage meta: %w", err)
	}

	if err := coverage.WriteCountersDir(dir); err != nil {
		return fmt.Errorf("write coverage counters: %w", err)
	}

	return nil
}
