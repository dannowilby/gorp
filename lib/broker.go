package gorp

import "sync"

type LogEntry struct {
	Term    int
	Message string
}

type Broker struct {
	Role Role

	ChangeLock sync.Mutex
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

func (broker *Broker) ChangeRole(role Role) {
	// lock so that other goroutines/the RPCs cannot run
	broker.ChangeLock.Lock()
	broker.Role = role
	broker.ChangeLock.Unlock()
}

func (broker *Broker) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {

	// Make sure we're not changing, might want to check that this doesn't get
	// reduced reduced to a noop
	broker.ChangeLock.Lock()
	broker.ChangeLock.Unlock()

	return broker.Role.RequestVote(msg, rply)
}

func (broker *Broker) AppendMessage(am AppendMessage, mr *AppendMessageReply) error {

	// Make sure we're not changing, might want to check that this doesn't get
	// reduced reduced to a noop
	broker.ChangeLock.Lock()
	broker.ChangeLock.Unlock()

	return broker.Role.AppendMessage(am, mr)
}

func (broker *Broker) Execute() (Role, error) {
	return broker.Role.Execute()
}

// Potentially common RPCs between role types:
// - A config change entails the following process
//   1. augment the current config with the new config
//   2. commit the augmented config as a log entry
//   3. now that it is commited, change to the new config
