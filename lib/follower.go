package gorp

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
)

type Follower struct {
	State *State
}

/* RPC methods */

func (follower Follower) AppendMessage(message string) {
	fmt.Println(follower.State.commit_index)
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
