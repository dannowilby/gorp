package gorp_role

import (
	"context"
	"fmt"
	"math/rand"
	"net/rpc"
	"strconv"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Candidate struct {
	State *gorp.State

	VoteTimeoutSignal chan Broker
	RPCSignal         chan Broker
}

func (candidate *Candidate) Init(state *gorp.State) Broker {

	candidate.State = state
	candidate.VoteTimeoutSignal = make(chan Broker, 1)
	candidate.RPCSignal = make(chan Broker, 1)

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

	select {
	case next_role := <-candidate.VoteTimeoutSignal:
		return next_role, nil
	case next_role := <-candidate.RPCSignal:
		return next_role, nil
	}
}

func (candidate Candidate) SendRequest(ctx context.Context, vote_status chan bool, element string) {

	client, err := rpc.DialHTTP("tcp", element)

	if err != nil {
		vote_status <- false
		return
	}

	request_vote_args := gorp_rpc.RequestVoteMessage{
		Term:        candidate.State.CommitTerm,
		CandidateId: candidate.State.Host,

		LastLogIndex: candidate.State.CommitIndex,
		LastLogTerm:  candidate.State.CommitTerm,
	}
	request_vote_rply := gorp_rpc.RequestVoteReply{}

	call := make(chan *rpc.Call, 1)

	client.Go("Broker.RequestVote", request_vote_args, &request_vote_rply, call)

	select {
	case <-ctx.Done():
		vote_status <- false
	case <-call:
		vote_status <- request_vote_rply.VoteGranted
	}

}

func (candidate *Candidate) Execute(ctx context.Context) {

	// Transitioning to this state, we immediately update the term
	candidate.State.CommitTerm += 1

	// create timeouts, election timeout is standard, timeout after the election is random
	election_timeout_duration, t1_err := time.ParseDuration(strconv.Itoa(candidate.State.ElectionTimeout) + "ms")
	timeout_duration, t2_err := time.ParseDuration(strconv.Itoa(rand.Intn(candidate.State.ElectionTimeout)) + "ms")
	if t1_err != nil || t2_err != nil {
		// have yet to implement proper error handling
		panic("please implement proper error handling please")
	}

	timeout_ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// vote counting mechanism
	votes := make(chan bool)

	total_votes := len(candidate.State.Config) - 1
	votes_needed := (total_votes / 2) + 1
	vote_tally := 0

	// call RequestVote to all other machines
	for _, element := range candidate.State.Config {

		if element == candidate.State.Host {
			continue
		}

		fmt.Println("Do we send a request to", element)
		// request vote from each machine in config
		go candidate.SendRequest(timeout_ctx, votes, element)

	}

	// wait for votes to come in
	// we could use a loop label, but I don't like those
	completed := false
	for completed {
		select {
		case <-time.After(election_timeout_duration):
			// if we've timed out, break out of the requests
			// start a new timeout, and then return a new candidate if no new
			// leader has been elected otherwise
			completed = true

		case vote := <-votes:
			// add the vote to the tally
			if vote {
				vote_tally += 1
			}

			// if we have enough votes, break and become leader
			completed = vote_tally >= votes_needed
		}
	}

	// for some reason, we decided that either we have enough, or we've timed out
	cancel()

	fmt.Println(vote_tally, votes_needed)

	// if a majority accept, then transition to a leader and sends heartbeats to
	// enforce its authority
	if vote_tally >= votes_needed {
		candidate.VoteTimeoutSignal <- new(Leader).Init(candidate.State)
		return
	}

	// if a majority does not occur, then time out, start a new term trying again
	<-time.After(timeout_duration)
	candidate.VoteTimeoutSignal <- new(Candidate).Init(candidate.State)
}

func (candidate *Candidate) Serve(ctx context.Context) {

}

func (candidate *Candidate) GetState() *gorp.State {
	return candidate.State
}
