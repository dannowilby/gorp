package gorp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Follower struct {
	State *State

	// add a mutex so the different go routines handling the RPC
	// endpoints can update the timeout
	last_request_lock sync.Mutex
	last_request      time.Time

	ChangeSignal chan *RoleTransition
}

func (follower *Follower) Init(state *State) *Follower {

	follower.State = state
	follower.State.Role = "follower"
	follower.State.VotedFor = ""

	// it's crucial that these channels are buffered so we don't wait for a
	// receiver to be ready, causing a deadlock
	follower.ChangeSignal = make(chan *RoleTransition, 1)

	return follower
}

func (follower *Follower) AppendMessage(message AppendMessage, reply *AppendMessageReply) error {

	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	if !AppendMessageIsUpToDate(follower.State, &message) || !PrevLogsMatch(follower.State, &message) {
		reply.CommitTerm = follower.State.CommitTerm
		reply.Success = false
		return nil
	}

	follower.State.Leader = message.LeaderId

	// if this isn't a standard heartbeat, then update log
	empty_log := LogEntry{Term: -2}
	if message.Entry.Term != empty_log.Term {

		// the previous message matches, now append the new messages, removing any
		// existing logs with conflicting index
		follower.State.Log = follower.State.Log[0:(message.PrevLogIndex + 1)]
		follower.State.Log = append(follower.State.Log, message.Entry)
	}

	// set commit index
	if message.LeaderCommit > follower.State.CommitIndex {
		// we update our commit index
		follower.State.CommitIndex = min(message.LeaderCommit, len(follower.State.Log)-1)

		follower.Apply()
	}

	follower.State.CommitTerm = message.Term

	reply.CommitTerm = follower.State.CommitTerm
	reply.Success = true

	return nil
}

func (follower *Follower) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {

	slog.Debug("Request received on follower!")

	if !CanVoteFor(follower.State, &msg) || !VoteMsgIsUpToDate(follower.State, &msg) {
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
	leaderClientHost := HostToClientHost(follower.State.Leader)
	if leaderClientHost == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "leader not known"})
		return
	}
	http.Redirect(w, r, leaderClientHost+r.RequestURI, http.StatusTemporaryRedirect)
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
				follower.ChangeSignal <- &RoleTransition{RoleName: "candidate", State: follower.State}
			}

			slog.Debug("Follower clock tick", "elapsed", elapsed)
		}
	}

}

func (follower *Follower) GetChangeSignal() chan *RoleTransition {
	return follower.ChangeSignal
}

func (follower *Follower) GetState() *State {
	return follower.State
}

func (follower *Follower) Apply() {
	log := follower.State.Log
	up_to := follower.State.CommitIndex
	last_applied := follower.State.LastApplied

	for last_applied != up_to {
		entry := log[last_applied+1]

		if entry.Type == "data" {
			fmt.Println("applying:", last_applied+1)
		}
		if entry.Type == "config" {
			fmt.Println("Updating config.")

			var config ConfigData

			// should not error due to previous error checking
			err := json.Unmarshal(entry.Message, &config)

			if err != nil {
				fmt.Println(err)
			}

			// update the config
			follower.State.Config = append(config.New, config.Old...)
		}

		last_applied++
		follower.State.LastApplied = last_applied
	}
}
