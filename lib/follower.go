package gorp

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Follower struct {
	State *State

	// add a mutex so the different go routines handling the RPC
	// endpoints can update the timeout
	last_request_lock sync.Mutex
	last_request      time.Time
}

func (follower *Follower) AppendMessage(message AppendMessage, reply *AppendMessageReply) error {

	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	// check that the leader term is acceptable
	if message.term < follower.State.commit_term {
		reply.CommitTerm = follower.State.commit_term
		reply.Success = false
		return nil
	}

	// if the log at the previous index does not contain the same term, then
	// return false
	if message.PrevLogIndex != -1 {
		if len(follower.State.log)-1 < message.PrevLogIndex || follower.State.log[message.prev_log_term].Term != message.prev_log_term {
			reply.CommitTerm = follower.State.commit_term
			reply.Success = false
			return nil
		}
	}

	// the previous message matches, now append the new messages, removing any
	// existing logs with conflicting index
	follower.State.log = follower.State.log[0:(message.PrevLogIndex + 1)]
	follower.State.log = append(follower.State.log, message.Entry)

	// set commit index
	if message.leader_commit > follower.State.commit_index {
		follower.State.commit_index = min(message.leader_commit, len(follower.State.log)-1)
	}

	reply.CommitTerm = follower.State.commit_term
	reply.Success = true
	return nil
}

func (follower *Follower) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {

	// msg not new enough
	if msg.term < follower.State.commit_term {
		rply.term = follower.State.commit_term
		rply.vote_granted = false
		return nil
	}

	// check that this machine has not voted for a different one this term
	if follower.State.voted_for != "" && follower.State.voted_for != msg.candidate_id {
		rply.term = follower.State.commit_term
		rply.vote_granted = false
		return nil
	}

	log := follower.State.log

	// check if its up-to-date
	if len(log) > 0 && msg.last_log_index < follower.State.last_applied || msg.last_log_term < log[len(log)-1].Term {
		rply.term = follower.State.commit_term
		rply.vote_granted = false
		return nil
	}

	// grant the vote
	rply.term = follower.State.commit_term
	rply.vote_granted = true

	// since successful, update the election timeout
	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	return nil
}

func monitorHeartbeat(ctx context.Context, cancel context.CancelFunc, follower *Follower) {
	dur, _ := time.ParseDuration("100ms") // time between ticker updates
	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			follower.last_request_lock.Lock()
			elapsed := time.Since(follower.last_request)
			follower.last_request_lock.Unlock()

			// multiple by 1000000 to convert ns to ms
			if elapsed > time.Duration(follower.State.ElectionTimeout*1000000) {
				cancel()
			}

			slog.Debug("Follower clock tick", "elapsed", elapsed)
		}
	}
}

func (follower *Follower) Execute() (Role, error) {

	// the replica has just converted from a candidate to a follower, so this is
	// because of a request that was sent
	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitorHeartbeat(ctx, cancel, follower)

	<-ctx.Done()

	candidate := new(Candidate)
	candidate.State = follower.State

	return candidate, nil
}

func (follower *Follower) GetState() *State {
	return follower.State
}
