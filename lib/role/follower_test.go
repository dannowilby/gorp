package gorp_role

import (
	"testing"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

func TestCanAppendMessageOnStartup(t *testing.T) {
	state := gorp.State{ElectionTimeout: 500}
	role := Follower{State: &state}

	msg := gorp_rpc.AppendMessage{PrevLogIndex: -1, Entry: gorp.LogEntry{Term: 0, Message: "simple message"}}

	rply := gorp_rpc.AppendMessageReply{}

	err := role.AppendMessage(msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage function.", err)
	}

	if !rply.Success {
		t.Fatal("Unsuccessful appending message.")
	}

	if len(state.Log) != 1 || state.Log[0].Message != "simple message" {
		t.Fatal("Logs don't match.")
	}
}

// Test that a disagreement between the leader and the follower, where the
// follower has an uncommited log from a previous turn, overwrites the log with
// the proper one
func TestCanAppendMessageFollowerOneAhead(t *testing.T) {

	logs := make([]gorp.LogEntry, 0)
	logs = append(logs, gorp.LogEntry{Term: 0, Message: "Existing 1."})
	logs = append(logs, gorp.LogEntry{Term: 0, Message: "Existing 2."})

	state := gorp.State{Log: logs, CommitIndex: 1, LastApplied: 1}
	role := Follower{State: &state}

	state.CommitTerm = 1

	msg := gorp_rpc.AppendMessage{Term: 1, PrevLogIndex: 0, PrevLogTerm: 0, Entry: gorp.LogEntry{Term: 1, Message: "simple message"}}

	rply := gorp_rpc.AppendMessageReply{}

	err := role.AppendMessage(msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage function.", err)
	}

	if rply.CommitTerm != 1 || !rply.Success {
		t.Fatal("AppendMessage failed.")
	}

	if state.Log[0].Message != "Existing 1." || state.Log[1].Message != "simple message" {
		t.Fatal("Logs don't match.")
	}
}

// Similar to the test above, this test makes sure that the replica does not
// overwrite its log as it is too far ahead. This should signal to the leader to
// decrement its prev_log counter for the replica and try calling AppendMessage again
func TestCanAppendMessageFollowerMultipleAhead(t *testing.T) {

	logs := make([]gorp.LogEntry, 0)
	logs = append(logs, gorp.LogEntry{Term: 0, Message: "Existing 1."})
	logs = append(logs, gorp.LogEntry{Term: 0, Message: "Existing 2."})

	state := gorp.State{Log: logs, CommitIndex: 1, LastApplied: 1}
	role := Follower{State: &state}

	state.CommitTerm = 1

	msg := gorp_rpc.AppendMessage{Term: 4, PrevLogIndex: 10, PrevLogTerm: 4, Entry: gorp.LogEntry{Term: 4, Message: "simple message"}}

	rply := gorp_rpc.AppendMessageReply{}

	err := role.AppendMessage(msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage function.", err)
	}

	if rply.CommitTerm != 1 || rply.Success {
		t.Fatal("AppendMessage should not have succeeded.")
	}

	if state.Log[0].Message != "Existing 1." || state.Log[1].Message != "Existing 2." {
		t.Fatal("Logs don't match.")
	}
}

func TestCanVote(t *testing.T) {
	state := gorp.State{}
	role := new(Follower).Init(&state)

	rqvtmsg := gorp_rpc.RequestVoteMessage{Term: 1}
	rqvtrply := gorp_rpc.RequestVoteReply{}

	role.RequestVote(rqvtmsg, &rqvtrply)

	if !rqvtrply.VoteGranted {
		t.Fatal("Vote not granted")
	}
}

// Test that the follower will reject the vote if it is more up-to-date than the
// one requesting
func TestCanRejectVote(t *testing.T) {
	state := gorp.State{CommitTerm: 2}
	role := new(Follower).Init(&state)

	rqvtmsg := gorp_rpc.RequestVoteMessage{Term: 1}
	rqvtrply := gorp_rpc.RequestVoteReply{}

	role.RequestVote(rqvtmsg, &rqvtrply)

	if rqvtrply.VoteGranted {
		t.Fatal("Vote granted")
	}
}

func TestWontChangeVote(t *testing.T) {
	state := gorp.State{}
	role := new(Follower).Init(&state)

	rqvtmsg := gorp_rpc.RequestVoteMessage{CandidateId: "machine_1:1234", Term: 1}
	rqvtrply := gorp_rpc.RequestVoteReply{}

	role.RequestVote(rqvtmsg, &rqvtrply)

	// the grant should be voted for the first request
	if !rqvtrply.VoteGranted {
		t.Fatal("Vote not granted")
	}

	// now we send another request, from a different machine

	rqvtmsg = gorp_rpc.RequestVoteMessage{CandidateId: "machine_2:1234", Term: 1}
	rqvtrply = gorp_rpc.RequestVoteReply{}

	role.RequestVote(rqvtmsg, &rqvtrply)

	if rqvtrply.VoteGranted {
		t.Fatal("Should not change vote")
	}
}

// Test to see if the vote is rejected if the logs themselves are not up-to-date
func TestVotingVerifiesLogs(t *testing.T) {
	logs := make([]gorp.LogEntry, 0)
	logs = append(logs, gorp.LogEntry{Term: 0, Message: "Existing 1."})
	logs = append(logs, gorp.LogEntry{Term: 0, Message: "Existing 2."})

	state := gorp.State{Log: logs, CommitIndex: 1, LastApplied: 1}
	role := Follower{State: &state}

	rqvtmsg := gorp_rpc.RequestVoteMessage{Term: 1, LastLogIndex: -1, LastLogTerm: 0}
	rqvtrply := gorp_rpc.RequestVoteReply{}

	role.RequestVote(rqvtmsg, &rqvtrply)

	if rqvtrply.VoteGranted {
		t.Fatal("Vote not granted")
	}
}
