package gorp

import (
	"errors"
)

type Candidate struct {
	State *State
}

// If this machine has already requested votes, then do nothing
func (candidate *Candidate) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {
	return nil
}

// Turn to follower if term is equal to or less than the message term
func (candidate *Candidate) AppendMessage(msg AppendMessage, rply *AppendMessageReply) error {
	return nil
}

func (candidate *Candidate) Execute() (Role, error) {

	// Transitioning to this state, we immediately update the term
	candidate.State.commit_term += 1

	// call RequestVote to all other machines, voting for itself

	for _, element := range candidate.State.config {

		if element == candidate.State.host {
			continue
		}

		// request vote from each machine in config
	}

	// if a majority accept, then transition to a leader and sends heartbeats to
	// enforce its authority

	// if a majority does not occur, then time out, start a new term trying again

	return nil, errors.New("not fully implemented")
}

func (candidate Candidate) GetState() *State {
	return candidate.State
}
