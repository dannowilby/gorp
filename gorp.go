package main

import (
	"fmt"

	gorp "github.com/dannowilby/gorp/lib"
)

func run() error {

	gorp.Hello()

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}
