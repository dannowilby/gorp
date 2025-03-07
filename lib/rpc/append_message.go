package gorp_rpc

import gorp "github.com/dannowilby/gorp/lib"

type AppendMessage struct {
	Term     int
	LeaderId string

	// -1 indicates that the log is empty
	PrevLogIndex int

	PrevLogTerm int
	Entry       gorp.LogEntry

	LeaderCommit int
}

type AppendMessageReply struct {
	CommitTerm int
	Success    bool
}
