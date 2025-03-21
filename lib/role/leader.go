package gorp_role

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Leader struct {
	State *gorp.State

	// essentially a queue of messages that we have to try and append
	msgs chan gorp.LogEntry

	// the next log index to send to the associated server
	nextIndex map[string]int

	// index of highest log entry known to be replicated on the server
	matchIndex map[string]int
}

func (leader *Leader) Init(state *gorp.State) Role {
	leader.State = state
	leader.State.Role = "leader"
	leader.msgs = make(chan gorp.LogEntry, 10)

	for _, server := range state.Config {
		leader.nextIndex[server] = len(leader.State.Log) - 1
		leader.matchIndex[server] = 0
	}

	return leader
}

func (leader *Leader) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {
	return nil
}

func (leader *Leader) AppendMessage(msg gorp_rpc.AppendMessage, rply *gorp_rpc.AppendMessageReply) error {
	return nil
}

func (leader *Leader) SendHeartbeats(ctx context.Context) {

	send_heartbeat := func() {
		for _, element := range leader.State.Config {

			if element == leader.State.Host {
				continue
			}

			// send heartbeat, we don't really care about what the response is right
			// now, but in the future it will be used to inductively get the
			// followers up to date

			client, err := rpc.DialHTTPPath("tcp", element, "/")

			if err != nil {
				fmt.Println(err)
				return
			}

			append_message_args := gorp_rpc.AppendMessage{Term: leader.State.CommitTerm, LeaderId: leader.State.Host}
			append_message_rply := gorp_rpc.AppendMessageReply{}
			client.Go("Broker.AppendMessage", append_message_args, &append_message_rply, nil)
		}
	}

	for {
		select {
		// after around half the time an election timeout takes, send a heartbeat
		case <-time.After(250 * time.Millisecond):
			send_heartbeat()
		case <-ctx.Done():
			return
		}
	}

}

func (leader *Leader) replicate(msg gorp.LogEntry) {

	replicated := false

	// add the msg to the leaders log

	for !replicated {

		for _, element := range leader.State.Config {

			// we don't need to send the msg to ourself
			if element == leader.State.Host {
				continue
			}

			// if the follower has all the logs, we don't need to send anything
			if len(leader.State.Log) == leader.matchIndex[element] {
				continue
			}

			// now we do need to send something

			// send the log located in nextIndex

			// if it comes back as a failure, decrement and try again

		}

	}

	// apply to this state machine

}

func (leader *Leader) Execute(ctx context.Context) {

	// start continuously heartbeating
	go leader.SendHeartbeats(ctx)

	for {
		select {
		case msg := <-leader.msgs:
			leader.replicate(msg)
		case <-ctx.Done():
			return
		}
	}

}

func (leader *Leader) NextRole(ctx context.Context) (Role, error) {

	<-ctx.Done()

	return nil, errors.New("cancelled")
}

func (leader *Leader) GetState() *gorp.State {
	return leader.State
}
