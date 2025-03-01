package gorp

import (
	"net"
	"net/http"
	"net/rpc"
)

type LogEntry struct {
	Term    int
	Message string
}

type Broker struct {
	Role Role
}

type RequestVoteMessage struct {
	term         int
	candidate_id string

	last_log_index int
	last_log_term  int
}

type RequestVoteReply struct {
	term int

	vote_granted bool
}

type AppendMessage struct {
	term      int
	leader_id string

	// -1 indicates that the log is empty
	PrevLogIndex int

	prev_log_term int
	Entry         LogEntry

	leader_commit int
}

type AppendMessageReply struct {
	CommitTerm int
	Success    bool
}

type Role interface {
	RequestVote(RequestVoteMessage, *RequestVoteReply) error

	AppendMessage(AppendMessage, *AppendMessageReply) error

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

func (broker *Broker) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {
	return broker.Role.RequestVote(msg, rply)
}

func (broker *Broker) AppendMessage(am AppendMessage, mr *AppendMessageReply) error {
	return broker.Role.AppendMessage(am, mr)
}

func (broker *Broker) Execute() {
	rpc.Register(broker)
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		return
	}

	broker.Role.Execute()

	http.Serve(l, nil)

}

// Potentially common RPCs between role types:
// - A config change entails the following process
//   1. augment the current config with the new config
//   2. commit the augmented config as a log entry
//   3. now that it is commited, change to the new config
