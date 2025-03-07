package gorp

import "context"

type Broker struct {
	Role Role
}

type Role interface {
	RequestVote(RequestVoteMessage, *RequestVoteReply) error

	AppendMessage(AppendMessage, *AppendMessageReply) error

	NextRole() (Role, error)

	// Used to implement the actual logic of the replicas/individual roles
	// Returns the next role to transition to
	// TODO: add context to here so that it can listen to cancel
	Execute()

	// Used as a type check to ensure that the role has a relation
	// to the underlying state object of the replica
	GetState() *State
}

func (broker *Broker) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {
	return broker.Role.RequestVote(msg, rply)
}

func (broker *Broker) AppendMessage(am AppendMessage, mr *AppendMessageReply) error {
	return broker.Role.AppendMessage(am, mr)
}

func (broker *Broker) Execute(ctx context.Context) {
	// pass in the context so it knows when to cancel
	broker.Role.Execute()
}

func (broker *Broker) Serve(ctx context.Context) {

	// start up and serve the RPC server

	<-ctx.Done()

	// shut down the RPC server
}

// Potentially common RPCs between role types:
// - A config change entails the following process
//   1. augment the current config with the new config
//   2. commit the augmented config as a log entry
//   3. now that it is commited, change to the new config
