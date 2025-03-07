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
