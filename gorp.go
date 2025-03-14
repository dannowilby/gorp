package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_role "github.com/dannowilby/gorp/lib/role"
)

func Run(ctx context.Context, state *gorp.State) error {

	var replica gorp_role.Broker = gorp_role.FromState(state)

	for {

		ctx, cancel := context.WithCancel(ctx)

		replica.StartServer()

		// start executing replica housekeeping
		go replica.Execute(ctx)

		// wait for a change of state
		next_role, err := replica.NextRole(ctx)

		// shutdown server and execution thread
		replica.StopServer()
		cancel()

		// if it is changing to shutdown or an error happened,
		// just return the err msg
		if err != nil {
			return err
		}

		// switch finally
		replica.SwitchRole(next_role)
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

	state := gorp.State{ElectionTimeout: 500}

	if err := Run(context.Background(), &state); err != nil {
		slog.Error("error, exiting", "error", err)
	} else {
		slog.Info("Shutting down gracefully.")
	}
}
