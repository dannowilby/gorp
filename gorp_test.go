package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"testing"

	gorp "github.com/dannowilby/gorp/lib"
)

type Scenario struct {
	Replicas []gorp.State `json:"replicas"`
	Expected []gorp.State `json:"expected"`
}

// Tests the interactions between different role types. This is not implemented
// in the normal Golang style as these tests need to be executed sequentially
// due to the need for them to create multiple HTTP listeners on the same ports.
func TestScenarios(t *testing.T) {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Set the logger as the default
	slog.SetDefault(logger)

	// scenarios, err := load_scenarios()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // for _, scenario := range scenarios {
	// // 	// run the scenario
	// // 	run_scenario(t, &scenario)

	// // }

}

func run_scenario(t *testing.T, scenario *Scenario) {

	ctx, cancel := context.WithCancel(context.Background())

	for i := range scenario.Replicas {

		fmt.Println(&scenario.Replicas[i])

		go Run(ctx, &scenario.Replicas[i])
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	<-c
	fmt.Println("\n\nGracefully shutting down...")
	for i := range scenario.Replicas {
		fmt.Println(&scenario.Replicas[i])
	}

}

// Read the scenario json files from `tests` and parse them
func load_scenarios() ([]Scenario, error) {
	file, err := os.Open("tests")
	if err != nil {
		return nil, err
	}
	raw_scenarios, err := file.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	fmt.Println(raw_scenarios)

	var scenarios []Scenario

	for _, rs := range raw_scenarios {
		f, err := os.Open("tests/" + rs)
		if err != nil {
			fmt.Println(err)
			continue
		}

		var scenario Scenario
		jsonParser := json.NewDecoder(f)
		if err := jsonParser.Decode(&scenario); err != nil {
			fmt.Println(err)
			continue
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}
