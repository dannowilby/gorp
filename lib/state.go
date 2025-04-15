package gorp

import (
	"encoding/json"
	"strconv"
	"strings"
)

type LogEntry struct {
	Term int `json:"term"`

	// Type specifies what type of message this should be interpreted as. This
	// may be a config change, update to storage, or potential snapshot data
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

type State struct {
	// used to redirect clients to the appropriate leader
	Leader string `json:"leader"`

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

func EmptyState() State {
	return State{
		Host:              "localhost:1234",
		Role:              "follower",
		ElectionTimeout:   500,
		RandomizedTimeout: []int{150, 300},
		Config:            []string{"localhost:1234"},
		CommitTerm:        -1,
		CommitIndex:       -1,
		LastApplied:       -1,
	}
}

// Get the number of machines needed for a majority, does remove the calling
// machine from the majority. AKA the number of separate machines needed for a majority.
func NumMajority(state *State) int {
	return ((len(state.Config) + 1) / 2) - 1
}

func HostToClientHost(host string) string {
	segments := strings.Split(host, ":")
	port, err := strconv.Atoi(segments[1])
	if err != nil {
		return ""
	}
	return "http://" + segments[0] + ":" + strconv.Itoa(port+3000)
}
