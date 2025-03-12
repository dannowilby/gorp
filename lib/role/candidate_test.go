package gorp_role

import (
	"context"
	"reflect"
	"testing"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
)

func TestCanTimeout(t *testing.T) {

}

// We test that the vote is not accepted if the candidate's term is not acceptable
func TestCanHandleTermRejectionVotes(t *testing.T) {
	ctx := context.Background()

	election_timeout := 500

	candidate_state := gorp.State{CommitTerm: 1, Host: "unimportant", Config: []string{"localhost:1234"}, ElectionTimeout: election_timeout}
	follower_state := gorp.State{CommitTerm: 2, Host: "localhost:1234"}

	var candidate Broker = new(Candidate).Init(&candidate_state)
	var follower Broker = new(Follower).Init(&follower_state)

	go follower.Serve(ctx)

	// wait for the follower server to start
	duration, _ := time.ParseDuration("50ms")
	<-time.After(duration)

	candidate.Execute(ctx)

	next_role, err := candidate.NextRole()

	if err != nil {
		t.Fatal(err)
	}

	if reflect.TypeOf(next_role) != reflect.TypeFor[*Candidate]() {
		t.Fatal("incorrect state change", reflect.TypeOf(next_role))
	}
}

func TestCanWinElection(t *testing.T) {

}

func TestCanLoseElection(t *testing.T) {

}
