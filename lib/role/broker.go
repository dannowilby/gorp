package gorp_role

import (
	"context"
	"strings"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Broker interface {
	Init(*gorp.State) Broker

	RequestVote(gorp_rpc.RequestVoteMessage, *gorp_rpc.RequestVoteReply) error

	AppendMessage(gorp_rpc.AppendMessage, *gorp_rpc.AppendMessageReply) error

	NextRole(context.Context) (Broker, error)

	// Used to implement the actual logic of the replicas/individual roles
	// Returns the next role to transition to
	Execute(context.Context)

	// Starts the RPC server
	Serve(context.Context)

	// Used as a type check to ensure that the role has a relation
	// to the underlying state object of the replica
	GetState() *gorp.State
}

func FromState(state *gorp.State) Broker {
	switch strings.ToLower(state.Role) {
	case "leader":
		return new(Leader).Init(state)
	case "candidate":
		return new(Candidate).Init(state)
	default:
		return new(Follower).Init(state)
	}
}

// Potentially common RPCs between role types:
// - A config change entails the following process
//   1. augment the current config with the new config
//   2. commit the augmented config as a log entry
//   3. now that it is commited, change to the new config
