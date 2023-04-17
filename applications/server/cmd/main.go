package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/sync/errgroup"

	"github.com/donmikel/karma8/applications/server"
	"github.com/donmikel/karma8/applications/server/adapters/inmemory"
	"github.com/donmikel/karma8/applications/server/config"
	"github.com/donmikel/karma8/applications/server/handlers/http"
	"github.com/donmikel/karma8/applications/server/interfaces"
	"github.com/donmikel/karma8/applications/server/services"
)

// exitCode is a process termination code.
type exitCode int

// Possible process termination codes are listed below.
const (
	// exitSuccess is code for successful program termination.
	exitSuccess exitCode = 0
	// exitFailure is code for unsuccessful program termination.
	exitFailure exitCode = 1
)

const storageCount = 7

// Kubernetes (rolling update) doesn't wait until a pod is out of rotation before sending SIGTERM,
// and external LB could still route traffic to a non-existing pod resulting in a surge of 50x API errors.
// It's recommended to wait for 5 seconds before terminating the program; see references
// https://github.com/kubernetes-retired/contrib/issues/1140, https://youtu.be/me5iyiheOC8?t=1797.
const preStopWait = 5 * time.Second

// Shutdown timeout for http servers.
const shutdownTimeout = 5 * time.Second

var (
	// version is the service version from git tag.
	version = ""
)

func main() {
	os.Exit(int(gracefulMain()))
}

// gracefulMain releases resources gracefully upon termination.
// When we call os.Exit defer statements do not run resulting in unclean process shutdown.
// nolint
func gracefulMain() exitCode {
	var logger log.Logger
	{
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configPath := fs.String("config", "", "path to the config file")
	v := fs.Bool("v", false, "Show version")

	err := fs.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		return exitSuccess
	}
	if err != nil {
		logger.Log("msg", "parsing cli flags failed", "err", err)
		return exitFailure
	}

	if *v {
		if version == "" {
			level.Error(logger).Log("Version not set")
		} else {
			level.Info(logger).Log("Version: %s\n", version)
		}

		return exitSuccess
	}

	logger.Log("configPath", *configPath)

	cfg, err := config.Parse(*configPath)
	if err != nil {
		logger.Log("msg", "cannot parse service config", "err", err)
		return exitFailure
	}

	err = cfg.Validate()
	if err != nil {
		logger.Log("msg", "config validation failed", "err", err)
		return exitFailure
	}

	// It's nice to be able to see panics in Logs, hence we monitor for panics after
	// logger has been bootstrapped.
	defer monitorPanic(logger)
	ctx := context.Background()

	var fileMetaStorage interfaces.FileMetaStorage
	{
		fileMetaStorage = inmemory.NewFileMetaStorage()
	}

	var storageManager interfaces.StorageManager
	{
		storageManager = inmemory.NewStorageManager(logger)
	}

	for i := 0; i < storageCount; i++ {
		storageURL := fmt.Sprintf("storage_%d", i)
		err = storageManager.AddStorage(ctx, storageURL, inmemory.NewStorage(storageURL, logger))
		if err != nil {
			level.Error(logger).Log("msg", "error adding storage",
				"err", err,
			)

			return exitFailure
		}
	}

	var fileService server.FileService
	{
		fileService = services.NewService(fileMetaStorage, storageManager)
	}

	hServer := http.NewHTTPServer(cfg.API, fileService, logger)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case s := <-sig:
			level.Info(logger).Log("msg", fmt.Sprintf("signal received (waiting %v before terminating): %v", preStopWait, s))
			time.Sleep(preStopWait)
			level.Info(logger).Log("msg", "terminating...")

			return fmt.Errorf("signal received: %s", s)
		}
	})

	group.Go(func() error {
		if err := hServer.ListenAndServe(); err != nil {
			return fmt.Errorf("listen and server error: %w", err)
		}
		return nil
	})

	group.Go(func() error {
		<-ctx.Done()

		level.Info(logger).Log("msg", "graceful shutdown of server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err = hServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}

		return ctx.Err()
	})

	if err = group.Wait(); err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("actors stopped with err: %v", err))
		return exitFailure
	}

	level.Info(logger).Log("msg", "actors stopped without errors")

	return exitSuccess
}

// monitorPanic monitors panics and reports them somewhere (e.g. logs, ...).
func monitorPanic(logger log.Logger) {
	if rec := recover(); rec != nil {
		err := fmt.Sprintf("panic: %v \n stack trace: %s", rec, debug.Stack())
		level.Error(logger).Log("err", err)
		panic(err)
	}
}
