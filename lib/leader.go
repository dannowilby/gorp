package gorp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Leader struct {
	State *State

	// essentially a queue of messages that we have to try and append
	MessageQueue chan LogEntry

	// the next log index to send to the associated server
	nextIndex map[string]int

	// index of highest log entry known to be replicated on the server
	matchIndex map[string]int

	ChangeSignal chan *RoleTransition
}

func (leader *Leader) Init(state *State) *Leader {
	leader.State = state
	leader.State.Role = "leader"

	leader.MessageQueue = make(chan LogEntry, 10)
	leader.ChangeSignal = make(chan *RoleTransition, 1)

	leader.nextIndex = make(map[string]int)
	leader.matchIndex = make(map[string]int)

	for _, server := range state.Config {
		leader.nextIndex[server] = len(leader.State.Log)
		leader.matchIndex[server] = 0
	}

	fmt.Println("Leader:", leader.State.Host)

	return leader
}

func (leader *Leader) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {
	if VoteMsgIsUpToDate(leader.State, &msg) {
		rply.VoteGranted = true
		rply.Term = leader.State.CommitTerm
		return nil
	}

	rply.VoteGranted = false
	rply.Term = leader.State.CommitTerm

	return nil
}

func (leader *Leader) AppendMessage(msg AppendMessage, rply *AppendMessageReply) error {
	// we need to check if the length of the logs is longer and if the term is
	// the same, if true, then
	if !AppendMessageIsNewer(leader.State, &msg) {

		// the msg sender has a longer log, therefore it should be the leader
		if AppendMessageLogIsLonger(leader.State, &msg) {
			leader.ChangeSignal <- &RoleTransition{RoleName: "follower", State: leader.State}
		}

		rply.CommitTerm = leader.State.CommitTerm
		rply.Success = false
		return nil
	}

	// unlike the follower, we don't modify anything else
	// this allows all the behavior that handles log synchronization by the
	// follower role

	rply.CommitTerm = leader.State.CommitTerm
	rply.Success = true

	leader.ChangeSignal <- &RoleTransition{RoleName: "follower", State: leader.State}

	return nil
}

func (leader *Leader) HandleClient(w http.ResponseWriter, r *http.Request) {
	var message LogEntry
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		w.WriteHeader(400)
	}
	message.Term = leader.State.CommitTerm

	// if this is a config change,
	// preprocess so that c_old_new is commited.
	if message.Type == "config" {

		var config ConfigData

		err := json.Unmarshal(message.Message, &config)

		if err != nil {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode("Invalid configuration format.")
			return
		}

		config.Old = leader.State.Config
		fmt.Println(message)
	}

	leader.MessageQueue <- message
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("Message queued to be saved.")
}

func (leader *Leader) Execute(ctx context.Context) {

	// start continuously heartbeating
	go leader.SendHeartbeats(ctx)

	for {
		select {
		// when we have a message that needs replicating
		case msg := <-leader.MessageQueue:
			leader.Replicate(ctx, msg)
		case <-ctx.Done():
			return
		}
	}

}

func (leader *Leader) GetChangeSignal() chan *RoleTransition {
	return leader.ChangeSignal
}

func (leader *Leader) GetState() *State {
	return leader.State
}
