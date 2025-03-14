package gorp_role

import (
	"testing"
)

func TestCanTimeout(t *testing.T) {

}

// // We test that the vote is not accepted if the candidate's term is not acceptable
// func TestCanHandleTermRejectionVotes(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())

// 	election_timeout := 500

// 	candidate_state := gorp.State{CommitTerm: 1, Host: "unimportant", Config: []string{"localhost:1234"}, ElectionTimeout: election_timeout}
// 	follower_state := gorp.State{CommitTerm: 2, Host: "localhost:1234"}

// 	var candidate Broker = new(Candidate).Init(&candidate_state)
// 	var follower Broker = new(Follower).Init(&follower_state)

// 	go follower.Serve(ctx)

// 	// wait for the follower server to start
// 	duration, _ := time.ParseDuration("50ms")
// 	<-time.After(duration)

// 	candidate.Execute(ctx)

// 	next_role, err := candidate.NextRole()

// 	cancel()

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if reflect.TypeOf(next_role) != reflect.TypeFor[*Candidate]() {
// 		t.Fatal("incorrect state change", reflect.TypeOf(next_role))
// 	}
// }

// func TestCanWinElection(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())

// 	election_timeout := 500

// 	candidate_state := gorp.State{CommitTerm: 2, Host: "unimportant", Config: []string{"localhost:1234", "localhost:1235"}, ElectionTimeout: election_timeout}
// 	follower_a_state := gorp.State{CommitTerm: 1, Host: "localhost:1234", ElectionTimeout: election_timeout}
// 	// follower_b_state := gorp.State{CommitTerm: 1, Host: "localhost:1235", ElectionTimeout: election_timeout}

// 	var candidate Broker = new(Candidate).Init(&candidate_state)
// 	var follower_a Broker = new(Follower).Init(&follower_a_state)
// 	// var follower_b Broker = new(Follower).Init(&follower_b_state)

// 	go follower_a.Serve(ctx)
// 	// go follower_b.Serve(ctx)

// 	// wait for the follower server to start
// 	duration, _ := time.ParseDuration("50ms")
// 	<-time.After(duration)

// 	candidate.Execute(ctx)

// 	next_role, err := candidate.NextRole()

// 	cancel()

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if reflect.TypeOf(next_role) != reflect.TypeFor[*Candidate]() {
// 		t.Fatal("incorrect state change", reflect.TypeOf(next_role))
// 	}
// }

func TestCanLoseElection(t *testing.T) {

}
