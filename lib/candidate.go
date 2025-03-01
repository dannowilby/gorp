package gorp

import (
	"errors"
)

type Candidate struct {
	State *State
}

func (candidate *Candidate) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {
	return nil
}

func (candidate *Candidate) AppendMessage(msg AppendMessage, rply *AppendMessageReply) error {
	return nil
}

func (candidate *Candidate) Execute() (Role, error) {

	// Transitioning to this state, we immediately update the term
	candidate.State.commit_term += 1

	// call RequestVote to all other machines, voting for itself

	// if a majority accept, then transition to a leader and sends heartbeats to
	// enforce its authority

	// if a majority does not occur, then time out, start a new term trying again

	return nil, errors.New("not fully implemented")
}

func (candidate Candidate) GetState() *State {
	return candidate.State
}
