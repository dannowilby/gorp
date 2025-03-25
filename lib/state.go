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
	// Index 0 is the minimum timeout, and index 1 is the maximum, used in
	// candidate replicas to calculate how long they should time out for after
	// losing an election
	RandomizedTimeout []int `json:"randomizedTimeout"`
}

// Get the number of machines needed for a majority, does remove the calling
// machine from the majority. AKA the number of separate machines needed for a majority.
func NumMajority(state *State) int {
	return ((len(state.Config) + 1) / 2) - 1
}
