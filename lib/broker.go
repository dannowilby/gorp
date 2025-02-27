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
	// Returns the next role to transition to
	Execute() (Role, error)

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

	// timeout in milliseconds
	ElectionTimeout int
}

// Potentially common RPCs between role types:
// - A config change entails the following process
//   1. augment the current config with the new config
//   2. commit the augmented config as a log entry
//   3. now that it is commited, change to the new config
