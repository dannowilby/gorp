package gorp

type AppendMessage struct {
	term      int
	leader_id string

	// -1 indicates that the log is empty
	PrevLogIndex int

	prev_log_term int
	Entry         LogEntry

	leader_commit int
}

type AppendMessageReply struct {
	CommitTerm int
	Success    bool
}
