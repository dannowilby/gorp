package gorp_role

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
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

func (leader *Leader) SendHeartbeats(ctx context.Context) {

	send_heartbeat := func() {
		for _, element := range leader.State.Config {

			if element == leader.State.Host {
				continue
			}

			// send heartbeat, we don't really care about what the response is right
			// now, but in the future it will be used to inductively get the
			// followers up to date

			client, err := rpc.DialHTTPPath("tcp", element, "/")

			if err != nil {
				fmt.Println(err)
				return
			}

			append_message_args := gorp_rpc.AppendMessage{
				Term:         leader.State.CommitTerm,
				LeaderId:     leader.State.Host,
				PrevLogIndex: -1,
				PrevLogTerm:  -1,
				LeaderCommit: leader.State.CommitIndex,
				Entry:        gorp.LogEntry{Term: -2},
			}
			append_message_rply := gorp_rpc.AppendMessageReply{}
			client.Go("Broker.AppendMessage", append_message_args, &append_message_rply, nil)
		}
	}

	for {
		select {
		// after around half the time an election timeout takes, send a heartbeat
		case <-time.After(250 * time.Millisecond):
			send_heartbeat()
		case <-ctx.Done():
			return
		}
	}

}

// Updates the machine and tries to append
func (leader *Leader) append_to_machine(ctx context.Context, success chan bool, host string, msg gorp.LogEntry) {

	client, err := rpc.DialHTTPPath("tcp", host, "/")

	if err != nil {
		fmt.Println(err)
		success <- false
		return
	}

	last_log := leader.State.CommitIndex
	msg_to_send := leader.nextIndex[host]

	up_to_date := last_log < 0 || last_log+1 == msg_to_send
	for !up_to_date {

		prev_index := msg_to_send - 1
		prev_term := -1
		if prev_index > -1 {
			prev_term = leader.State.Log[prev_index].Term
		}

		append_message_args := gorp_rpc.AppendMessage{
			Term:     leader.State.CommitTerm,
			LeaderId: leader.State.Host,

			PrevLogTerm:  prev_term,
			PrevLogIndex: prev_index,

			Entry: leader.State.Log[msg_to_send],

			LeaderCommit: last_log,
		}

		append_message_rply := gorp_rpc.AppendMessageReply{}
		err := client.Call("Broker.AppendMessage", append_message_args, &append_message_rply)

		if err != nil {
			success <- false
			fmt.Println(err)
			return
		}

		// update the next log to send
		if append_message_rply.Success {
			leader.nextIndex[host]++
		} else {
			leader.nextIndex[host]--
		}
		msg_to_send = leader.nextIndex[host]

		// check that we are up-to-date
		up_to_date = last_log+1 == leader.nextIndex[host]
	}

	prev_term := -1
	prev_index := msg_to_send - 1

	if prev_index > -1 {
		prev_term = leader.State.Log[prev_index].Term
	}

	append_message_args := gorp_rpc.AppendMessage{
		Term:     leader.State.CommitTerm,
		LeaderId: leader.State.Host,

		PrevLogTerm:  prev_term,
		PrevLogIndex: prev_index,

		Entry: msg,

		LeaderCommit: last_log,
	}
	append_message_rply := gorp_rpc.AppendMessageReply{}

	err = client.Call("Broker.AppendMessage", append_message_args, &append_message_rply)
	if err != nil {
		success <- false
		fmt.Println(err)
		return
	}

	// if the message was successfully replicated, update the counter for the
	// machine, so for the next message we don't resend it
	if append_message_rply.Success {
		leader.nextIndex[host]++
	}

	// send if it was successsful or not
	success <- append_message_rply.Success
}

// In its current form, this implementation may cause issues, cancelling
// normally functioning appends when a majority is achieved. This needs further
// testing, but it should still offer certainty that a majority has replicated
// any one log message.
func (leader *Leader) replicate(parent_ctx context.Context, msg gorp.LogEntry) {

	ctx, cancel := context.WithCancel(parent_ctx)

	// query machines
	accepted := make(chan bool)
	for _, host := range leader.State.Config {
		if host == leader.State.Host {
			continue
		}

		// send message to machines, get response through accepted channel
		go leader.append_to_machine(ctx, accepted, host, msg)
	}

	majority := gorp.NumMajority(leader.State)
	vote_count := 0

	for {
		select {
		case success := <-accepted:
			if success {
				vote_count++
			}

			if vote_count > majority {

				// append and apply the log
				leader.State.Log = append(leader.State.Log, msg)
				leader.State.CommitIndex += 1

				// stop all the machines from trying to update,
				// if a machine is unable to be updated in the allotted time
				// before a majority, then the next time a message comes in it can
				// have another go, with hopefully an already more up-to-date
				// system
				cancel()
				return
			}

		case <-ctx.Done():
			cancel()
			return
		}
	}

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

func (leader *Leader) NextRole(ctx context.Context) (Role, error) {

	select {
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	case next_role := <-leader.ChangeSignal:
		return next_role, nil
	}
}

func (leader *Leader) GetState() *gorp.State {
	return leader.State
}
