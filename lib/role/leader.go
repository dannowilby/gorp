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
}

func (leader *Leader) Init(state *gorp.State) Role {
	leader.State = state
	leader.State.Role = "leader"

	return leader
}

func (leader *Leader) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {
	return nil
}

func (leader *Leader) AppendMessage(msg gorp_rpc.AppendMessage, rply *gorp_rpc.AppendMessageReply) error {
	return nil
}

func (leader *Leader) SendHeartbeats() {

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

func (leader *Leader) Execute(ctx context.Context) {

	for {
		select {
		// after around half the time an election timeout takes, send a heartbeat
		case <-time.After(250 * time.Millisecond):
			go leader.SendHeartbeats()
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
