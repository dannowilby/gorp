package gorp

type LogEntry struct {
	Term    int
	Message string
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
