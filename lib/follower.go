package gorp

import (
	"net"
	"net/http"
	"net/rpc"
)

type Follower struct {
	State *State
}

/* RPC methods */

type AppendMessage struct {
	term           int
	leader_id      string
	prev_log_index int

	prev_log_term int
	entry         LogEntry

	leader_commit int
}

func (follower Follower) AppendMessage(message AppendMessage) (int, bool) {

	// check that the terms are the same
	if message.term != follower.State.commit_term {
		return follower.State.commit_term, false
	}

	// check if the log contains the prev_index
	if message.prev_log_index >= len(follower.State.log) {
		return follower.State.commit_term, false // Warning, think about the inductive case! This will cause errors!
	}

	// now that we know prev_index exists, check that it has the correct term
	if message.prev_log_term != follower.State.log[message.prev_log_index].term {
		return follower.State.commit_term, false
	}

	// the previous message matches, now append the new messages, removing any
	// existing logs with conflicting index

	// set commit index

	return 0, true
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
	http.Serve(l, nil)

	return follower
}

func (follower Follower) GetState() *State {
	return follower.State
}
