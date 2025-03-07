package gorp

type LogEntry struct {
	Term    int
	Message string
}

type State struct {

	// the running replica's host/port
	Host string

	// the set of servers participating in consensus
	Config []string

	// persistent state
	Log        []LogEntry
	CommitTerm int
	VotedFor   string

	// volatile state
	CommitIndex int
	LastApplied int

	// timeout in milliseconds
	ElectionTimeout int
}
