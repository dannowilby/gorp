package gorp

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
)

type Candidate struct {
	State *State

	ChangeSignal chan *RoleTransition
}

func (candidate *Candidate) Init(state *State) *Candidate {

	slog.Debug("[candidate] creating new candidate state")

	candidate.State = state
	candidate.State.Role = "candidate"
	candidate.ChangeSignal = make(chan *RoleTransition, 1)
	candidate.State.VotedFor = PeerAddress{}

	return candidate
}

func (candidate *Candidate) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {

	slog.Debug("[candidate] request vote received", "vote_for", msg.CandidateId)

	if !CanVoteFor(candidate.State, &msg) || !VoteMsgIsUpToDate(candidate.State, &msg) {

		slog.Debug("[candidate] request vote not new enough", "vote_for", msg.CandidateId, "can_vote_for", CanVoteFor(candidate.State, &msg), "vote_msg_uptodate", VoteMsgIsUpToDate(candidate.State, &msg))

		rply.Term = candidate.State.CommitTerm
		rply.VoteGranted = false
		return nil
	}

	slog.Debug("[candidate] request vote granteded", "vote_for", msg.CandidateId)

	candidate.State.VotedFor = msg.CandidateId

	// grant the vote
	rply.Term = candidate.State.CommitTerm
	rply.VoteGranted = true

	return nil
}

// Another machine has turned into a leader. It has done so with a majority of
// votes, so now this machine should also accept the leader.
// Turn to follower if term is equal to or less than the message term
func (candidate *Candidate) AppendMessage(message AppendMessage, reply *AppendMessageReply) error {

	slog.Debug("[candidate] append message received")

	if !AppendMessageIsUpToDate(candidate.State, &message) {

		slog.Debug("[candidate] append message not new enough", "up_to_date", AppendMessageIsUpToDate(candidate.State, &message))

		reply.CommitTerm = candidate.State.CommitTerm
		reply.Success = false
		return nil
	}

	slog.Debug("[candidate] message newer, converting to follower")

	// unlike the follower, we don't modify anything else
	// this allows all the behavior that handles log synchronization by the
	// follower role
	reply.CommitTerm = candidate.State.CommitTerm
	reply.Success = false

	candidate.ChangeSignal <- &RoleTransition{RoleName: "follower", State: candidate.State}

	return nil
}

func (candidate *Candidate) SendRequest(ctx context.Context, vote_status chan bool, element string) {

	client, err := rpc.DialHTTPPath("tcp", element, "/")

	if err != nil {
		fmt.Println(err)
		vote_status <- false
		return
	}

	request_vote_args := RequestVoteMessage{
		Term:        candidate.State.CommitTerm,
		CandidateId: candidate.State.PeerAddress,

		LastLogIndex: candidate.State.CommitIndex,
		LastLogTerm:  candidate.State.CommitTerm,
	}
	request_vote_rply := RequestVoteReply{}
	call := client.Go("Broker.RequestVote", request_vote_args, &request_vote_rply, nil)

	select {
	case <-ctx.Done():
		vote_status <- false
	case <-call.Done:
		vote_status <- request_vote_rply.VoteGranted
	}

}

func (candidate *Candidate) HandleClient(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func (candidate *Candidate) Execute(ctx context.Context) {

	slog.Debug("[candidate] starting candidate execution")

	// Transitioning to this state, we immediately update the term
	candidate.State.CommitTerm += 1

	// create initial randomized timeout
	timeout_duration, t2_err := time.ParseDuration(strconv.Itoa(
		candidate.State.RandomizedTimeout[0]+rand.Intn(candidate.State.RandomizedTimeout[1]-candidate.State.RandomizedTimeout[0])) + "ms")
	if t2_err != nil {
		// have yet to implement proper error handling
		panic("please implement proper error handling please, config is probably bad")
	}
	slog.Debug("[candidate] initial timeout starting", "duration", timeout_duration)
	<-time.After(timeout_duration)
	slog.Debug("[candidate] finished initial timeout")

	// if already voted for another machine, don't try gathering votes
	emptyAddress := PeerAddress{}
	if candidate.State.VotedFor != emptyAddress {
		slog.Debug("[candidate] voted during initial timeout", "voted_for", candidate.State.VotedFor)
		return
	}

	// vote for self
	candidate.State.VotedFor = candidate.State.PeerAddress
	slog.Debug("[candidate] voting for self")

	election_timeout_duration, t1_err := time.ParseDuration(strconv.Itoa(candidate.State.ElectionTimeout) + "ms")
	if t1_err != nil {
		// have yet to implement proper error handling
		panic("please implement proper error handling please, config is probably bad")
	}

	timeout_ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	slog.Debug("[candidate] preparing to gather votes")

	// vote counting mechanism
	votes := make(chan bool)

	votes_threshold := NumMajority(candidate.State)
	vote_tally := 1
	votes_received := 0

	// call RequestVote to all other machines
	for _, element := range candidate.State.Config {

		if element.RPCAddr() == candidate.State.PeerAddress.RPCAddr() {
			continue
		}

		slog.Debug("[candidate] gathering votes for self", "asking", element)

		// request vote from each machine in config
		go candidate.SendRequest(timeout_ctx, votes, element.RPCAddr())
	}

	// wait for votes to come in
	completed := false
	for !completed {
		select {
		case <-ctx.Done():
			slog.Debug("[candidate] execution cancelled")
			// cancelled for some reason
			cancel()
			return
		case <-time.After(election_timeout_duration):
			slog.Debug("[candidate] done collecting votes")
			// oh no, we've timed out
			completed = true

		case vote := <-votes:
			votes_received += 1

			if vote {
				vote_tally += 1
			}

			completed = vote_tally >= votes_threshold
		}
	}

	slog.Debug("[candidate] vote counts", "votes tally", vote_tally, "votes received", votes_received, "votes threshold", votes_threshold)

	cancel()

	// if a majority accept, then transition to a leader and sends heartbeats to
	// enforce its authority
	if vote_tally > votes_threshold {
		candidate.ChangeSignal <- &RoleTransition{RoleName: "leader", State: candidate.State}
		slog.Debug("[candidate] new leader elected", "elected_at", time.Now())
		return
	}

	slog.Debug("[candidate] did not receive enough votes, resetting candidate")

	candidate.ChangeSignal <- &RoleTransition{RoleName: "candidate", State: candidate.State}
}

func (candidate *Candidate) GetChangeSignal() chan *RoleTransition {
	return candidate.ChangeSignal
}

func (candidate *Candidate) GetState() *State {
	return candidate.State
}
