package gorp_rpc

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
