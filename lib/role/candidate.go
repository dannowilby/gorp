package gorp_role

import (
	"context"
	"errors"
	"log/slog"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Candidate struct {
	State *gorp.State
}

func (candidate *Candidate) Init(state *gorp.State) Broker {

	candidate.State = state

	return candidate
}

// If this machine has already requested votes, then do nothing
func (candidate *Candidate) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {
	return nil
}

// Turn to follower if term is equal to or less than the message term
func (candidate *Candidate) AppendMessage(msg gorp_rpc.AppendMessage, rply *gorp_rpc.AppendMessageReply) error {
	return nil
}

func (candidate *Candidate) NextRole() (Broker, error) {
	<-time.After(500 * time.Millisecond)
	return nil, errors.New("unimplemented")
}

func (candidate *Candidate) Execute(ctx context.Context) {
	slog.Debug("imagine as if this candidate role were implemented")

	// Transitioning to this state, we immediately update the term
	candidate.State.CommitTerm += 1

	// call RequestVote to all other machines, voting for itself

	for _, element := range candidate.State.Config {

		if element == candidate.State.Host {
			continue
		}

		// request vote from each machine in config
	}

	// if a majority accept, then transition to a leader and sends heartbeats to
	// enforce its authority

	// if a majority does not occur, then time out, start a new term trying again

}

func (candidate *Candidate) Serve(ctx context.Context) {

}

func (candidate *Candidate) GetState() *gorp.State {
	return candidate.State
}
