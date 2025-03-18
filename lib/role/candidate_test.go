package gorp_role

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

func TestCanAcceptNewLeader(t *testing.T) {
	state := gorp.State{CommitIndex: -1, LastApplied: -1}
	role := new(Candidate).Init(&state)

	msg := gorp_rpc.AppendMessage{Term: 1, PrevLogIndex: -1, PrevLogTerm: -1}

	rply := gorp_rpc.AppendMessageReply{}

	err := role.AppendMessage(msg, &rply)
	if err != nil {
		t.Fatal("Error calling AppendMessage function.", err)
	}

	if !rply.Success {
		t.Fatal("Did not submit")
	}

	var next chan Role

	go func() {
		n, err := role.NextRole(context.Background())
		fmt.Println(reflect.TypeOf(n))
		if err != nil {
			return
		}
		next <- n
	}()

	n, err := role.NextRole(context.Background())

	if err != nil {
		t.Fatal("Error changing roles", err)
	}

	if reflect.TypeOf(n) != reflect.TypeFor[*Follower]() {
		t.Fatal("Wrong next role")
	}

}
