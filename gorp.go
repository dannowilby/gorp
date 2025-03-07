package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"

	gorp "github.com/dannowilby/gorp/lib"
)

func run() error {

	state := gorp.State{ElectionTimeout: 500}
	role := gorp.Follower{State: &state}
	replica := gorp.Broker{Role: &role}

	for {

		ctx, cancel := context.WithCancel(context.Background())

		// start executing replica housekeeping
		go replica.Execute(ctx)

		// start replica RPC server
		go replica.Serve(ctx)

		// wait for a change of state
		next_role, err := replica.Role.NextRole()

		// shutdown server and execution thread
		cancel()

		// if it is changing to shutdown or an error happened,
		// just return the err msg
		if err != nil {
			return err
		}

		// switch finally
		replica.Role = next_role
	}
}

func configure_log() {
	logLevel := flag.String("lvl", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Set up the logger with the specified level
	var level slog.Level
	switch strings.ToLower(*logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create a logger with the desired level
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	// Set the logger as the default
	slog.SetDefault(logger)
}

func main() {
	configure_log()

	if err := run(); err != nil {
		slog.Error("Error, exiting.", "error", err)
	} else {
		slog.Info("Shutting down gracefully.")
	}
}
