package gorp

type RequestVoteMessage struct {
	Term        int
	CandidateId string

	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term int

	VoteGranted bool
}

/*
We may be facing issues with this function (maybe with the append message
functions too), because of the following excerpt from the Raft paper:
"Raft determines which of two logs is more up-to-date
by comparing the index and term of the last entries in the
logs. If the logs have last entries with different terms, then
the log with the later term is more up-to-date. If the logs
end with the same term, then whichever log is longer is
more up-to-date."
*/
func VoteMsgIsUpToDate(state *State, msg *RequestVoteMessage) bool {
	terms_match := msg.Term >= state.CommitTerm

	log := state.Log
	// we are currently only checking if msg term is >= than state,
	// we need to factor in log length
	logs_match := !(len(log) > 0 && (msg.LastLogIndex < state.LastApplied || msg.LastLogTerm < log[len(log)-1].Term))

	return terms_match && logs_match
}

func CanVoteFor(state *State, msg *RequestVoteMessage) bool {
	return !(state.VotedFor != "" && state.VotedFor != msg.CandidateId)
}
