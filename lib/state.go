package gorp

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type LogEntry struct {
	Term int `json:"term"`

	// Type specifies what type of message this should be interpreted as. This
	// may be a config change, update to storage, or potential snapshot data
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`

	// Hash is generated from the message content and used to track
	// the status of the entry. Persisted so a new leader can recover this.
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
}

type MessageData struct {
	Path      string `json:"path"`
	Blob      string `json:"blob"`
	Operation string `json:"operation"` // "write", "delete", "update"
}

// Response sent back to the client when they first submit a message
type SubmitResponse struct {
	Hash      string `json:"hash"`
	Timestamp string `json:"timestamp"`
}

// Response sent back when a client polls for status
type StatusResponse struct {
	Status string `json:"status"` // "success", "pending", "failed"
	Hash   string `json:"hash"`
}

type ConfigData struct {
	Old []string `json:"old"`
	New []string `json:"new"`
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

// RoleTransition represents a request to transition to a new role
type RoleTransition struct {
	RoleName string // "leader", "candidate", "follower", or "" for shutdown
	State    *State
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
