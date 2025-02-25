package main

import (
	"fmt"
	"reflect"

	gorp "github.com/dannowilby/gorp/lib"
)

func run() error {

	replica := gorp.Broker{
		State: gorp.State{},
		Role:  gorp.Candidate{},
	}

	for {
		next_role := replica.Run()
		fmt.Println("Next role:", reflect.TypeOf(next_role))

		switch next_role.(type) {
		case gorp.Exiting:
			return nil
		default:
			replica.Role = next_role
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Shut down gracefully.")
	}
}
