package gorp

import (
	"errors"
)

type Candidate struct {
	State *State
}

func (Candidate) Execute() (Role, error) {
	return nil, errors.New("Unimplemented!")
}

func (candidate Candidate) GetState() *State {
	return candidate.State
}
