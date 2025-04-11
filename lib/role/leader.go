package gorp_role

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

type Leader struct {
	State *gorp.State

	// essentially a queue of messages that we have to try and append
	msgs chan gorp.LogEntry

	// the next log index to send to the associated server
	nextIndex map[string]int

	// index of highest log entry known to be replicated on the server
	matchIndex map[string]int

	ChangeSignal chan Role
}

func (leader *Leader) Init(state *gorp.State) Role {
	leader.State = state
	leader.State.Role = "leader"

	leader.msgs = make(chan gorp.LogEntry, 10)
	leader.ChangeSignal = make(chan Role, 1)

	leader.nextIndex = make(map[string]int)
	leader.matchIndex = make(map[string]int)

	for _, server := range state.Config {
		leader.nextIndex[server] = len(leader.State.Log)
		leader.matchIndex[server] = 0
	}

	fmt.Println("Leader:", leader.State.Host)

	return leader
}

func (leader *Leader) RequestVote(msg gorp_rpc.RequestVoteMessage, rply *gorp_rpc.RequestVoteReply) error {
	if gorp_rpc.VoteMsgIsUpToDate(leader.State, &msg) {
		rply.VoteGranted = true
		rply.Term = leader.State.CommitTerm
		return nil
	}

	rply.VoteGranted = false
	rply.Term = leader.State.CommitTerm

	return nil
}

func (leader *Leader) AppendMessage(msg gorp_rpc.AppendMessage, rply *gorp_rpc.AppendMessageReply) error {
	if !gorp_rpc.AppendMessageIsNewer(leader.State, &msg) {
		rply.CommitTerm = leader.State.CommitTerm
		rply.Success = false
		return nil
	}

	// unlike the follower, we don't modify anything else
	// this allows all the behavior that handles log synchronization by the
	// follower role

	rply.CommitTerm = leader.State.CommitTerm
	rply.Success = true

	leader.ChangeSignal <- new(Follower).Init(leader.State)

	return nil
}

type ClientMessage struct {
	Message string `json:"message"`
}

func (leader *Leader) AcceptClientRequest(ctx context.Context) {
	rpc_port, err := strconv.Atoi(strings.Split(leader.State.Host, ":")[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	port := rpc_port + 3000

	// Create a simple handler that responds with a message
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var client_message ClientMessage
		err := json.NewDecoder(r.Body).Decode(&client_message)
		if err != nil {
			w.WriteHeader(400)
		}

		leader.msgs <- gorp.LogEntry{Term: leader.State.CommitTerm, Message: client_message.Message}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode("Message queued to be saved.")
	})

	// Configure the server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: nil, // Use the default ServeMux
	}

	// Start server in a goroutine so it doesn't block
	go func() {
		fmt.Printf("Server starting on port %d...\n", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

func (leader *Leader) Execute(ctx context.Context) {

	// start continuously heartbeating
	go leader.SendHeartbeats(ctx)

	// listen for user messages
	go leader.AcceptClientRequest(ctx)

	for {
		select {
		// when we have a message that needs replicating
		case msg := <-leader.msgs:
			leader.replicate(ctx, msg)
		case <-ctx.Done():
			return
		}
	}

}

func (leader *Leader) GetChangeSignal() chan Role {
	return leader.ChangeSignal
}

func (leader *Leader) GetState() *gorp.State {
	return leader.State
}
