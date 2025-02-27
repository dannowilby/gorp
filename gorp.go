package main

import (
	"fmt"
	"reflect"

	gorp "github.com/dannowilby/gorp/lib"
)

func run() error {

	state := gorp.State{ElectionTimeout: 500}
	role := gorp.Follower{State: &state}
	replica := gorp.Broker{Role: &role}

	for {
		next_role := replica.Role.Execute()
		fmt.Println("Next role:", reflect.TypeOf(next_role))

		switch next_role.(type) {
		case gorp.Exiting:
			return next_role.(gorp.Exiting).Error
		default:
			replica.Role = next_role
		}
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Shutting down gracefully.")
	}
}
