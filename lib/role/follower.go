package gorp_role

import (
	"context"
	"log/slog"
	"net/http"
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
	follower.State.VotedFor = ""

	// it's crucial that these channels are buffered so we don't wait for a
	// receiver to be ready, causing a deadlock
	follower.ChangeSignal = make(chan Role, 1)

	return follower
}

func (follower *Follower) AppendMessage(message gorp_rpc.AppendMessage, reply *gorp_rpc.AppendMessageReply) error {

	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	if !gorp_rpc.AppendMessageIsUpToDate(follower.State, &message) || !gorp_rpc.PrevLogsMatch(follower.State, &message) {
		reply.CommitTerm = follower.State.CommitTerm
		reply.Success = false
		return nil
	}

	// if this isn't a standard heartbeat, then update log
	empty_log := gorp.LogEntry{Term: -2}
	if message.Entry.Term != empty_log.Term {

		// the previous message matches, now append the new messages, removing any
		// existing logs with conflicting index
		follower.State.Log = follower.State.Log[0:(message.PrevLogIndex + 1)]
		follower.State.Log = append(follower.State.Log, message.Entry)

		// set commit index
		if message.LeaderCommit > follower.State.CommitIndex {
			// we update our commit index
			follower.State.CommitIndex = min(message.LeaderCommit, len(follower.State.Log)-1)
		}

	}

	reply.CommitTerm = follower.State.CommitTerm
	reply.Success = true

	follower.State.CommitTerm = message.Term
	// apply log to state machine, right now, this isn't implemented yet. This
	// might also require reversing applied messages if this replica had become
	// out of sync with the leader
	follower.State.LastApplied = follower.State.CommitIndex

	return nil
}

func (follower *Follower) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {

	slog.Debug("Request received on follower!")

	if !gorp_rpc.CanVoteFor(follower.State, &msg) || !gorp_rpc.VoteMsgIsUpToDate(follower.State, &msg) {
		rply.Term = follower.State.CommitTerm
		rply.VoteGranted = false
		return nil
	}

	// grant the vote
	rply.Term = follower.State.CommitTerm
	rply.VoteGranted = true

	// We need to update the votedFor member here
	follower.State.VotedFor = msg.CandidateId

	// since successful, update the election timeout
	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	return nil
}

func (follower *Follower) HandleClient(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
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

func (follower *Follower) GetChangeSignal() chan Role {
	return follower.ChangeSignal
}

func (follower *Follower) GetState() *gorp.State {
	return follower.State
}
