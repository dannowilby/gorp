package gorp

type LogEntry struct {
	Term    int    `json:"term"`
	Message string `json:"message"`
}

type State struct {

	// the running replica's host/port
	Host string `json:"host"`

	// mostly used for debugging and testing, stores a string representation of the current role
	Role string `json:"role"`

	// the set of servers participating in consensus
	Config []string `json:"config"`

	// persistent state
	Log        []LogEntry `json:"log"`
	CommitTerm int        `json:"commitTerm"`
	VotedFor   string     `json:"votedFor"`

	// volatile state
	CommitIndex int `json:"commitIndex"`
	LastApplied int `json:"lastApplied"`

	// timeout in milliseconds
	ElectionTimeout int `json:"electionTimeout"`
}
