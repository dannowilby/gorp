package gorp

import "errors"

type Leader struct {
	State *State
}

func (leader Leader) Execute() (Role, error) {

	return nil, errors.New("Unimplemented!")
}

func (leader Leader) GetState() *State {
	return leader.State
}
