package gorp

import "fmt"

type Candidate struct {
	State *State
}

func (Candidate) Execute() Role {
	fmt.Println("Running as candidate replica.")
	return Exiting{}
}

func (candidate Candidate) GetState() *State {
	return candidate.State
}
