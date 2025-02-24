package gorp

type log_entry struct {
	term    int
	message string
}

func follower() {
	// listen for append message rpcs to this replica
}

type Broker struct {
	// the set of servers participating in consensus
	config []string

	// persistent state
	log         []log_entry
	commit_term int
	voted_for   string

	// volatile state
	commit_index int
	last_applied int

	// the functionality to run if leader, candidate, or follower
	role_func func()
}

// a config change entails the following process
// 1. augment the current config with the new config
// 2. commit the augmented config as a log entry
// 3. now that it is commited, change to the new config
