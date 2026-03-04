package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	gorp "github.com/dannowilby/gorp/internal"
)

type Config struct {
	Replicas []gorp.State `json:"replicas"`
}

func Run(ctx context.Context, state *gorp.State) error {

	var replica gorp.Broker = gorp.FromState(state)

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
	ctx, cancel := context.WithCancel(context.Background())

	logLevel := flag.String("lvl", "info", "Log level (debug, info, warn, error)")
	id := flag.Int("id", -1, "The instance to run (-1 will start all instances of the config on the same machine)")
	config := flag.String("config", "", "Load cluster config from a JSON file (empty config will just start a candidate)")

	flag.Parse()

	gorp.ConfigureLog(logLevel)

	if *id > -1 && *config == "" {
		fmt.Println("Config must be set if id is not -1")
		return
	}

	var replicas *Config

	if *config != "" {
		loaded, err := load_config(*config)
		if err != nil {
			slog.Error("error, exiting", "error", err)
			return
		}
		replicas = loaded
	} else {
		replicas = &Config{Replicas: []gorp.State{gorp.EmptyState()}}
	}

	if *id < 0 {
		for i := range replicas.Replicas {

			// print the replicas initial state
			(&replicas.Replicas[i]).Debug_Print()

			go Run(ctx, &replicas.Replicas[i])
		}

	} else {
		// find a way to decide which replica in the config we are running
		(&replicas.Replicas[*id]).Debug_Print()
		go Run(ctx, &replicas.Replicas[*id])
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	<-c
	fmt.Println("\n\nExiting...")

	if *id < 0 {
		for i := range replicas.Replicas {
			(&replicas.Replicas[i]).Debug_Print()
		}
	} else {
		(&replicas.Replicas[*id]).Debug_Print()
	}

}
