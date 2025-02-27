package gorp

import (
	"net/rpc"
	"testing"
	"time"
)

func TestCanMakeAppendMessageRPC(t *testing.T) {
	state := State{ElectionTimeout: 500}
	role := Follower{State: &state}
	replica := Broker{Role: &role}

	go replica.Role.Execute()

	// wait for server to start
	time.Sleep(time.Millisecond * 100)

	client, err := rpc.DialHTTP("tcp", "localhost"+":1234")
	if err != nil {
		t.Fatal("Unable to connect to broker.", err)
	}

	msg := AppendMessage{PrevLogIndex: -1}

	rply := MessageReply{}

	err = client.Call("Follower.AppendMessage", &msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage RPC.", err)
	}

	t.Log(rply.Success, rply.CommitTerm)
}

func TestCanAppendMessageOnStartup(t *testing.T) {
	state := State{ElectionTimeout: 500}
	role := Follower{State: &state}

	msg := AppendMessage{PrevLogIndex: -1, Entry: LogEntry{Term: 0, Message: "simple message"}}

	rply := MessageReply{}

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
