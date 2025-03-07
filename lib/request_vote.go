package gorp

type RequestVoteMessage struct {
	term         int
	candidate_id string

	last_log_index int
	last_log_term  int
}

type RequestVoteReply struct {
	term int

	vote_granted bool
}
