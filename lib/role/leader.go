package gorp_role

import (
	"context"
	"errors"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Leader struct {
	State *gorp.State
}

func (leader *Leader) Init(state *gorp.State) Broker {
	leader.State = state

	return leader
}

func (leader *Leader) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {
	return nil
}

func (leader *Leader) AppendMessage(msg gorp_rpc.AppendMessage, rply *gorp_rpc.AppendMessageReply) error {
	return nil
}

func (leader *Leader) Execute(ctx context.Context) {

}

func (leader *Leader) Serve(ctx context.Context) {

}

func (leader *Leader) NextRole(ctx context.Context) (Broker, error) {

	<-ctx.Done()

	return nil, errors.New("cancelled")
}

func (leader *Leader) GetState() *gorp.State {
	return leader.State
}
