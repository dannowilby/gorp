package gorp

type Leader struct {
	State *State
}

func (leader Leader) Execute() Role {

	return Leader{}
}

func (leader Leader) GetState() *State {
	return leader.State
}
