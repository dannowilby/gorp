package gorp

import "errors"

type Leader struct {
	State *State
}

func (leader *Leader) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {
	return nil
}

func (leader *Leader) AppendMessage(msg AppendMessage, rply *AppendMessageReply) error {
	return nil
}

func (leader Leader) Execute() (Role, error) {

	return nil, errors.New("unimplemented")
}

func (leader Leader) GetState() *State {
	return leader.State
}
