package gorp

import (
	"encoding/json"
	"fmt"
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

type PeerAddress struct {
	Host     string `json:"host"`
	RPCPort  int    `json:"rpcport"`
	HTTPPort int    `json:"httpport"`
}

func (p PeerAddress) RPCAddr() string {
	return fmt.Sprintf("%s:%d", p.Host, p.RPCPort)
}

func (p PeerAddress) HTTPAddr() string {
	return fmt.Sprintf("http://%s:%d", p.Host, p.HTTPPort)
}

type ConfigData struct {
	Old []PeerAddress `json:"old"`
	New []PeerAddress `json:"new"`
}

type State struct {
	// used to redirect clients to the appropriate leader
	Leader PeerAddress `json:"leader"`

	// the running replica's host/port
	PeerAddress PeerAddress `json:"address"`

	// mostly used for debugging and testing, stores a string representation of the current role
	Role string `json:"role"`

	// the set of servers participating in consensus
	Config []PeerAddress `json:"config"`

	// persistent state
	Log        []LogEntry  `json:"log"`
	CommitTerm int         `json:"commitTerm"`
	VotedFor   PeerAddress `json:"votedFor"`

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

func (state *State) Debug_Print() {
	fmt.Println()
	fmt.Println("=======================================")
	fmt.Println()

	fmt.Println("INSTANCE STATE")
	fmt.Printf("%s\n", state.Role)
	fmt.Printf("%s:%d/%d\n\n", state.PeerAddress.Host, state.PeerAddress.RPCPort, state.PeerAddress.HTTPPort)

	fmt.Printf("Term %d\nIndex %d\nLast Applied %d\n", state.CommitTerm, state.CommitIndex, state.LastApplied)
	fmt.Printf("\nLog length: %d\n\n", len(state.Log))

	fmt.Println("=======================================")
	fmt.Println()
}

// RoleTransition represents a request to transition to a new role
type RoleTransition struct {
	RoleName string // "leader", "candidate", "follower", or "" for shutdown
	State    *State
}

func EmptyState() State {

	peerAddr := PeerAddress{
		Host:     "",
		RPCPort:  1234,
		HTTPPort: 4234,
	}

	return State{
		PeerAddress:       peerAddr,
		Role:              "follower",
		ElectionTimeout:   500,
		RandomizedTimeout: []int{150, 300},
		Config:            []PeerAddress{peerAddr},
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
