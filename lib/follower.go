package gorp

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
)

// Implements the AppendMessage RPC as defined in the Raft paper. The election
// timeout is kept track of by a separate go function which ticks down, checking
// the time of the last request. If the timeout has expired, then the replica is
// converted to a candidate.
type Follower struct {
	State *State

	// add a mutex so the different go routines handling the RPC
	// endpoints can update the timeout
	last_request_lock sync.Mutex
	last_request      time.Time
}

/* RPC methods */

type AppendMessage struct {
	term      int
	leader_id string

	// -1 indicates that the log is empty
	PrevLogIndex int

	prev_log_term int
	Entry         LogEntry

	leader_commit int
}

type MessageReply struct {
	CommitTerm int
	Success    bool
}

func (follower *Follower) AppendMessage(message AppendMessage, reply *MessageReply) error {

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

// func (follower *Follower) RequestVote() {

// }

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

/* Role setup/common methods */

func (follower *Follower) Execute() (Role, error) {

	// the replica has just converted from a candidate to a follower, so this is
	// because of a request that was sent
	follower.last_request_lock.Lock()
	follower.last_request = time.Now()
	follower.last_request_lock.Unlock()

	// register the RPC
	rpc.Register(follower)
	rpc.HandleHTTP()

	// set up the listener for incoming RPC requests
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitorHeartbeat(ctx, cancel, follower)

	go func() {
		http.Serve(l, nil)
	}()

	<-ctx.Done()
	l.Close()

	candidate := new(Candidate)
	candidate.State = follower.State

	return candidate, nil
}

func (follower *Follower) GetState() *State {
	return follower.State
}
