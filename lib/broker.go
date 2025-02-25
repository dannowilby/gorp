package gorp

type LogEntry struct {
	term    int
	message string
}

type Broker struct {
	Role Role
}

type Role interface {
	// Used to implement the actual logic of the replicas/individual roles
	Execute() Role

	// Used as a type check to ensure that the role has a relation
	// to the underlying state object of the replica
	GetState() *State
}

type State struct {

	// the running replica's host/port
	host string

	// the set of servers participating in consensus
	config []string

	// persistent state
	log         []LogEntry
	commit_term int
	voted_for   string

	// volatile state
	commit_index int
	last_applied int
}

type Exiting struct {
	State *State
	Error error
}

func (Exiting) Execute() Role {
	return Exiting{}
}

func (exiting Exiting) GetState() *State {
	return exiting.State
}

// a config change entails the following process
// 1. augment the current config with the new config
// 2. commit the augmented config as a log entry
// 3. now that it is commited, change to the new config
