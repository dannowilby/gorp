package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_role "github.com/dannowilby/gorp/lib/role"
)

type Config struct {
	Replicas []gorp.State `json:"replicas"`
}

func Run(ctx context.Context, state *gorp.State) error {

	var replica gorp_role.Broker = gorp_role.FromState(state)

	for {

		ctx, cancel := context.WithCancel(ctx)

		replica.StartRPCServer()
		replica.StartClientServer()

		// start executing replica housekeeping
		go replica.Execute(ctx)

		// wait for a change of state
		next_role, err := replica.NextRole(ctx)

		// shutdown server and execution thread
		replica.StopClientServer()
		replica.StopRPCServer()

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

func load_config(path string) (*Config, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var replicas Config
	jsonParser := json.NewDecoder(f)
	if err := jsonParser.Decode(&replicas); err != nil {
		return nil, err
	}

	return &replicas, nil
}

func main() {
	configure_log()

	local := flag.Bool("local", true, "Run the config locally")
	config := flag.String("config", "config.local.json", "Load cluster config from a JSON file")

	var replicas *Config

	if *config != "" {
		loaded, err := load_config(*config)
		if err != nil {
			slog.Error("error, exiting", "error", err)
			return
		}
		replicas = loaded
	}

	ctx, cancel := context.WithCancel(context.Background())

	if *local {
		for i := range replicas.Replicas {

			// print the replicas initial state
			fmt.Println(&replicas.Replicas[i])

			go Run(ctx, &replicas.Replicas[i])
		}

	} else {
		// find a way to decide which replica in the config we are running
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	<-c
	fmt.Println("\n\nExiting...")

	if *local {
		for i := range replicas.Replicas {
			fmt.Println(&replicas.Replicas[i])
		}
	} else {
		// only print the host replica
	}

}
