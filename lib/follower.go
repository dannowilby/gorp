package gorp

import (
	"context"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
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
}

/* RPC methods */

type AppendMessage struct {
	term      int
	leader_id string

	// -1 indicates that the log is empty
	prev_log_index int

	prev_log_term int
	entry         LogEntry

	leader_commit int
}

type MessageReply struct {
	commit_term int
	success     bool
}

func (follower Follower) AppendMessage(message AppendMessage, reply *MessageReply) error {

	// reset timeout
	follower.State.elapsed_timeout = follower.State.election_timeout

	// check that the terms are the same
	if message.term != follower.State.commit_term {
		reply.commit_term = follower.State.commit_term
		reply.success = false
		return nil
	}

	// check if the log contains the prev_index
	if message.prev_log_index >= len(follower.State.log) && message.prev_log_index != -1 {
		reply.commit_term = follower.State.commit_term
		reply.success = false
		return nil // Warning, think about the inductive case! This will cause errors!
	}

	// now that we know prev_index exists, check that it has the correct term
	if message.prev_log_term != follower.State.log[message.prev_log_index].term {
		reply.commit_term = follower.State.commit_term
		reply.success = false
		return nil
	}

	// the previous message matches, now append the new messages, removing any
	// existing logs with conflicting index
	follower.State.log = follower.State.log[0:(message.prev_log_index + 1)]
	follower.State.log = append(follower.State.log, message.entry)

	// set commit index
	if message.leader_commit > follower.State.commit_index {
		follower.State.commit_index = min(message.leader_commit)
	}

	reply.commit_term = follower.State.commit_term
	reply.success = true
	return nil
}

func monitorHeartbeat(ctx context.Context, follower *Follower) {
	dur, _ := time.ParseDuration(strconv.Itoa(follower.State.election_timeout) + "ms")
	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	// check ticker.C for elapsed time
}

/* Role setup/common methods */

func (follower Follower) Execute() Role {

	// register the RPC
	rpc.Register(follower)
	rpc.HandleHTTP()

	// set up the listener for incoming RPC requests
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		return Exiting{State: follower.State, Error: err}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitorHeartbeat(ctx, &follower)

	go func() {
		http.Serve(l, nil)
	}()

	// when we get the done signal,
	// switch the states
	<-ctx.Done()
	l.Close()
	return Candidate{State: follower.State}
}

func (follower Follower) GetState() *State {
	return follower.State
}
