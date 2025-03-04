package gorp

import (
	"testing"
)

func TestCanAppendMessageOnStartup(t *testing.T) {
	state := State{ElectionTimeout: 500}
	role := Follower{State: &state}

	msg := AppendMessage{PrevLogIndex: -1, Entry: LogEntry{Term: 0, Message: "simple message"}}

	rply := AppendMessageReply{}

	err := role.AppendMessage(msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage function.", err)
	}

	if !rply.Success {
		t.Fatal("Unsuccessful appending message.")
	}

	if len(state.log) != 1 || state.log[0].Message != "simple message" {
		t.Fatal("Logs don't match.")
	}
}

// Test that a disagreement between the leader and the follower, where the
// follower has an uncommited log from a previous turn, overwrites the log with
// the proper one
func TestCanAppendMessageFollowerOneAhead(t *testing.T) {

	logs := make([]LogEntry, 0)
	logs = append(logs, LogEntry{Term: 0, Message: "Existing 1."})
	logs = append(logs, LogEntry{Term: 0, Message: "Existing 2."})

	state := State{log: logs, commit_index: 1, last_applied: 1}
	role := Follower{State: &state}

	state.commit_term = 1

	msg := AppendMessage{term: 1, PrevLogIndex: 0, prev_log_term: 0, Entry: LogEntry{Term: 1, Message: "simple message"}}

	rply := AppendMessageReply{}

	err := role.AppendMessage(msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage function.", err)
	}

	if rply.CommitTerm != 1 || !rply.Success {
		t.Fatal("AppendMessage failed.")
	}

	if state.log[0].Message != "Existing 1." || state.log[1].Message != "simple message" {
		t.Fatal("Logs don't match.")
	}
}

// Similar to the test above, this test makes sure that the replica does not
// overwrite its log as it is too far ahead. This should signal to the leader to
// decrement its prev_log counter for the replica and try calling AppendMessage again
func TestCanAppendMessageFollowerMultipleAhead(t *testing.T) {

	logs := make([]LogEntry, 0)
	logs = append(logs, LogEntry{Term: 0, Message: "Existing 1."})
	logs = append(logs, LogEntry{Term: 0, Message: "Existing 2."})

	state := State{log: logs, commit_index: 1, last_applied: 1}
	role := Follower{State: &state}

	state.commit_term = 1

	msg := AppendMessage{term: 4, PrevLogIndex: 10, prev_log_term: 4, Entry: LogEntry{Term: 4, Message: "simple message"}}

	rply := AppendMessageReply{}

	err := role.AppendMessage(msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage function.", err)
	}

	if rply.CommitTerm != 1 || rply.Success {
		t.Fatal("AppendMessage should not have succeeded.")
	}

	if state.log[0].Message != "Existing 1." || state.log[1].Message != "Existing 2." {
		t.Fatal("Logs don't match.")
	}
}
