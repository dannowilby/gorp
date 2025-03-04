package main

import (
	"flag"
	"log/slog"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"reflect"
	"strings"

	gorp "github.com/dannowilby/gorp/lib"
)

func run() error {

	state := gorp.State{ElectionTimeout: 500}
	role := gorp.Follower{State: &state}
	replica := gorp.Broker{Role: &role}

	// start serving the broker's RPC calls
	// the calls themselves don't change, but the
	// underlying calls do

	rpc.Register(replica)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		return err
	}
	go http.Serve(l, nil)

	// Without locks, this is a recipe for disaster. If our server is sent a
	// message while the role is being changed, I would imagine that this causes
	// some sort of undefined behavior that we need to protect against.

	for {
		next_role, err := replica.Execute()

		if err != nil {
			l.Close()
			return err
		}

		slog.Info("Switching role.", "next_role", reflect.TypeOf(next_role))
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
