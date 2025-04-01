package gorp_role

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/rpc"
	"strconv"
	"sync"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Candidate struct {
	State *gorp.State

	Requesting sync.Mutex

	ChangeSignal chan Role
}

func (candidate *Candidate) Init(state *gorp.State) Role {

	candidate.State = state
	candidate.State.Role = "candidate"
	candidate.ChangeSignal = make(chan Role, 1)
	candidate.State.VotedFor = ""

	return candidate
}

// If this machine has already requested votes, then do nothing
func (candidate *Candidate) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {

	slog.Debug("Request received on candidate!")

	// if we are actively trying to get votes, don't allow it to vote for others
	if !candidate.Requesting.TryLock() {

		rply.Term = candidate.State.CommitTerm
		rply.VoteGranted = false

		return nil
	}
	// we are able to vote, so unlock
	candidate.Requesting.Unlock()

	if !gorp_rpc.CanVoteFor(candidate.State, &msg) || !gorp_rpc.VoteMsgIsUpToDate(candidate.State, &msg) {
		rply.Term = candidate.State.CommitTerm
		rply.VoteGranted = false
		return nil
	}

	// We have timed out and lost the election, now we should only vote for one
	// other machine
	candidate.State.VotedFor = msg.CandidateId

	// grant the vote
	rply.Term = candidate.State.CommitTerm
	rply.VoteGranted = true

	return nil
}

// Another machine has turned into a leader. It has done so with a majority of
// votes, so now this machine should also accept the leader.
// Turn to follower if term is equal to or less than the message term
func (candidate *Candidate) AppendMessage(message gorp_rpc.AppendMessage, reply *gorp_rpc.AppendMessageReply) error {

	if !gorp_rpc.AppendMessageIsUpToDate(candidate.State, &message) {
		reply.CommitTerm = candidate.State.CommitTerm
		reply.Success = false
		return nil
	}

	// unlike the follower, we don't modify anything else
	// this allows all the behavior that handles log synchronization by the
	// follower role

	reply.CommitTerm = candidate.State.CommitTerm
	reply.Success = true

	candidate.ChangeSignal <- new(Follower).Init(candidate.State)

	return nil
}

func (candidate *Candidate) NextRole(ctx context.Context) (Role, error) {

	select {
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	case next_role := <-candidate.ChangeSignal:
		return next_role, nil
	}
}

func (candidate *Candidate) SendRequest(ctx context.Context, vote_status chan bool, element string) {

	client, err := rpc.DialHTTPPath("tcp", element, "/")

	if err != nil {
		fmt.Println(err)
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
	call := client.Go("Broker.RequestVote", request_vote_args, &request_vote_rply, nil)

	select {
	case <-ctx.Done():
		vote_status <- false
	case <-call.Done:
		vote_status <- request_vote_rply.VoteGranted
	}

}

func (candidate *Candidate) Execute(ctx context.Context) {

	// we want to start the timeout before starting an election
	// and then vote for itself and try to gather votes after
	// if no other candidates want votes

	candidate.Requesting.Lock()

	// Transitioning to this state, we immediately update the term
	candidate.State.CommitTerm += 1

	// create timeouts, election timeout is standard, timeout after the election is random
	election_timeout_duration, t1_err := time.ParseDuration(strconv.Itoa(candidate.State.ElectionTimeout) + "ms")
	timeout_duration, t2_err := time.ParseDuration(strconv.Itoa(
		candidate.State.RandomizedTimeout[0]+rand.Intn(candidate.State.RandomizedTimeout[1]-candidate.State.RandomizedTimeout[0])) + "ms")
	if t1_err != nil || t2_err != nil {
		// have yet to implement proper error handling
		panic("please implement proper error handling please, config is probably bad")
	}

	timeout_ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// vote counting mechanism
	votes := make(chan bool)

	votes_needed := gorp.NumMajority(candidate.State)
	vote_tally := 1
	votes_received := 0

	// call RequestVote to all other machines
	for _, element := range candidate.State.Config {

		if element == candidate.State.Host {
			continue
		}

		// request vote from each machine in config
		go candidate.SendRequest(timeout_ctx, votes, element)
	}

	// wait for votes to come in
	// we could use a loop label, but I don't like those
	completed := false
	for !completed {
		select {
		case <-ctx.Done():
			// cancelled for some reason
			cancel()
			return
		case <-time.After(election_timeout_duration):
			// if we've timed out, break out of the requests
			// start a new timeout, and then return a new candidate if no new
			// leader has been elected otherwise
			completed = true

		case vote := <-votes:
			votes_received += 1
			// add the vote to the tally
			if vote {
				vote_tally += 1
			}

			// if we have enough votes, break and become leader
			// TODO: exit early if winning is impossible
			won_vote := vote_tally >= votes_needed

			completed = won_vote
		}
	}

	slog.Debug("vote counts", "votes tally", vote_tally, "votes received", votes_received, "votes needed", votes_needed)

	// for some reason, we decided that either we have enough, or we've timed out
	cancel()
	candidate.Requesting.Unlock()

	// if a majority accept, then transition to a leader and sends heartbeats to
	// enforce its authority
	if vote_tally >= votes_needed {
		candidate.ChangeSignal <- new(Leader).Init(candidate.State)
		return
	}

	// if a majority does not occur, then time out, start a new term trying again
	<-time.After(timeout_duration)
	candidate.ChangeSignal <- new(Candidate).Init(candidate.State)
}

func (candidate *Candidate) GetState() *gorp.State {
	return candidate.State
}
