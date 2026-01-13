package gorp

type AppendMessage struct {
	Term     int
	LeaderId string

	// -1 indicates that the log is empty
	PrevLogIndex int

	PrevLogTerm int
	Entry       LogEntry

	LeaderCommit int
}

type AppendMessageReply struct {
	CommitTerm int
	Success    bool
}

func AppendMessageIsNewer(state *State, msg *AppendMessage) bool {
	return msg.Term > state.CommitTerm
}

// when two leaders can't decide which one should be a leader, it resolves to
// which one has more committed messages
func AppendMessageLogIsLonger(state *State, msg *AppendMessage) bool {
	return msg.LeaderCommit > state.CommitIndex
}

// Compare the terms of the message and the machine.
func AppendMessageIsUpToDate(state *State, msg *AppendMessage) bool {
	return msg.Term >= state.CommitTerm
}

// If the log at the previous index does not contain the same term, then
// return false
func PrevLogsMatch(state *State, msg *AppendMessage) bool {

	if msg.PrevLogIndex != -1 {
		if len(state.Log)-1 < msg.PrevLogIndex ||
			state.Log[msg.PrevLogIndex].Term != msg.PrevLogTerm {
			return false
		}
	}

	return true
}
