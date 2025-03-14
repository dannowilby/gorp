package gorp_role

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Follower struct {
	State *gorp.State

	// add a mutex so the different go routines handling the RPC
	// endpoints can update the timeout
	last_request_lock sync.Mutex
	last_request      time.Time

	ChangeSignal chan Role
}

func (follower *Follower) Init(state *gorp.State) Role {

	follower.State = state
	follower.State.Role = "follower"

	// it's crucial that these channels are buffered so we don't wait for a
	// receiver to be ready, causing a deadlock
	follower.ChangeSignal = make(chan Role, 1)

	return follower
}

func (follower *Follower) AppendMessage(message gorp_rpc.AppendMessage, reply *gorp_rpc.AppendMessageReply) error {

	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	// check that the leader term is acceptable
	if message.Term < follower.State.CommitTerm {
		reply.CommitTerm = follower.State.CommitTerm
		reply.Success = false
		return nil
	}

	// if the log at the previous index does not contain the same term, then
	// return false
	if message.PrevLogIndex != -1 {
		if len(follower.State.Log)-1 < message.PrevLogIndex || follower.State.Log[message.PrevLogTerm].Term != message.PrevLogTerm {
			reply.CommitTerm = follower.State.CommitTerm
			reply.Success = false
			return nil
		}
	}

	// the previous message matches, now append the new messages, removing any
	// existing logs with conflicting index
	follower.State.Log = follower.State.Log[0:(message.PrevLogIndex + 1)]
	follower.State.Log = append(follower.State.Log, message.Entry)

	// TODO:
	// Update this machine's leader id
	// as AppendMessage establishes dominance

	// set commit index
	if message.LeaderCommit > follower.State.CommitIndex {
		follower.State.CommitIndex = min(message.LeaderCommit, len(follower.State.Log)-1)
	}

	reply.CommitTerm = follower.State.CommitTerm
	reply.Success = true
	return nil
}

func (follower *Follower) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {

	slog.Debug("Request received on follower!")

	// msg not new enough
	if msg.Term < follower.State.CommitTerm {
		rply.Term = follower.State.CommitTerm
		rply.VoteGranted = false
		return nil
	}

	// check that this machine has not voted for a different one this term
	if follower.State.VotedFor != "" && follower.State.VotedFor != msg.CandidateId {
		rply.Term = follower.State.CommitTerm
		rply.VoteGranted = false
		return nil
	}

	log := follower.State.Log

	// check if its up-to-date
	if len(log) > 0 && (msg.LastLogIndex < follower.State.LastApplied || msg.LastLogTerm < log[len(log)-1].Term) {
		rply.Term = follower.State.CommitTerm
		rply.VoteGranted = false
		return nil
	}

	// grant the vote
	rply.Term = follower.State.CommitTerm
	rply.VoteGranted = true

	// We need to update the votedFor member here

	// since successful, update the election timeout
	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	return nil
}

func (follower *Follower) NextRole(ctx context.Context) (Role, error) {
	select {
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	case next_role := <-follower.ChangeSignal:
		return next_role, nil
	}
}

// Checks on the heartbeat and turns into a candidate if no pulse
func (follower *Follower) Execute(ctx context.Context) {

	// the replica has just converted from a candidate to a follower, so this is
	// because of a request that was sent
	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	dur, _ := time.ParseDuration("100ms") // time between ticker updates
	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	for {
		select {
		// if context is cancelled, just return
		case <-ctx.Done():
			return
		// otherwise do heartbeat
		case <-ticker.C:
			follower.last_request_lock.Lock()
			elapsed := time.Since(follower.last_request)
			follower.last_request_lock.Unlock()

			// multiple by 1000000 to convert ns to ms
			if elapsed > time.Duration(follower.State.ElectionTimeout*1000000) {
				// if heartbeat fails, send new state to change to a new candidate
				follower.ChangeSignal <- new(Candidate).Init(follower.State)
			}

			slog.Debug("Follower clock tick", "elapsed", elapsed)
		}
	}

}

func (follower *Follower) GetState() *gorp.State {
	return follower.State
}
