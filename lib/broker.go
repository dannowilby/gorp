package gorp

import "fmt"

type log_entry struct {
	term    int
	message string
}

type Role interface {
	// sets up the RPC listener/other functionality
	// returns the next state of the replica
	Execute(State) Role
}

type State struct {
	// the set of servers participating in consensus
	config []string

	// persistent state
	log         []log_entry
	Commit_term int
	voted_for   string

	// volatile state
	commit_index int
	last_applied int
}

type Broker struct {
	State State
	Role  Role
}

func (broker Broker) Run() Role {
	return broker.Role.Execute(broker.State)
}

type Follower struct{}

func (Follower) Execute(state State) Role {
	return Follower{}
}

type Candidate struct{}

func (Candidate) Execute(state State) Role {
	fmt.Println("Running as candidate replica.")
	return Exiting{}
}

type Leader struct{}

func (Leader) Execute(state State) Role {
	return Leader{}
}

type Exiting struct{}

func (Exiting) Execute(state State) Role {
	return Exiting{}
}

// a config change entails the following process
// 1. augment the current config with the new config
// 2. commit the augmented config as a log entry
// 3. now that it is commited, change to the new config
