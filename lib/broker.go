package gorp

import "fmt"

// Broker type
const (
	Candidate = iota
	Follower
	Leader
)

type Broker struct {
	config string

	// persistent state
	log         []string
	commit_term int
	voted_for   string

	// volatile state
	commit_index int
	last_applied int
}

func Hello() {
	fmt.Println("Hello from broker!")
}
