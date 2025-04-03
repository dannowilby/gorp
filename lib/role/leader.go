package gorp_role

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"
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
		leader.nextIndex[server] = len(leader.State.Log) - 1
		leader.matchIndex[server] = 0
	}

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

	fmt.Println("appending")

	client, err := rpc.DialHTTPPath("tcp", host, "/")

	if err != nil {
		fmt.Println(err)
		return
	}

	// send/update machine
	// only send success when passed in log msg is replicated

	// we try updating automatically, and then after we know its up-to-date,
	// send the log

	last_log := leader.State.CommitIndex

	msg_to_send := leader.nextIndex[host]

	up_to_date := false
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
		fmt.Println("sending", msg_to_send, ",", append_message_rply.Success, ",", host, ",", prev_term, ",", prev_index)

		if err != nil {
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
	if msg_to_send != 0 {
		prev_term = leader.State.Log[msg_to_send-1].Term
	}

	append_message_args := gorp_rpc.AppendMessage{
		Term:     leader.State.CommitTerm,
		LeaderId: leader.State.Host,

		PrevLogTerm:  prev_term,
		PrevLogIndex: msg_to_send - 1,

		Entry: msg,

		LeaderCommit: last_log,
	}
	append_message_rply := gorp_rpc.AppendMessageReply{}

	err = client.Call("Broker.AppendMessage", append_message_args, &append_message_rply)
	if err != nil {
		return
	}

	// send if it was successsful
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
			fmt.Println("received vote back", success)
			if success {
				vote_count++
			}

			if vote_count > majority {
				// stop all the machines from trying to update,
				// if a machine is unable to be updated in the allotted time
				// before a majority, then the next time a message comes in it can
				// have another go, with hopefully an already more up-to-date
				// system

				// append and apply the log
				leader.State.Log = append(leader.State.Log, msg)
				leader.State.CommitIndex += 1

				cancel()
				return
			}

		case <-ctx.Done():
			cancel()
			return
		}
	}

}

func (leader *Leader) Execute(ctx context.Context) {

	// start continuously heartbeating
	go leader.SendHeartbeats(ctx)
	<-time.After(10 * time.Millisecond)
	leader.msgs <- gorp.LogEntry{Term: leader.State.CommitTerm, Message: "test"}

	for {
		select {
		// when we have a message that needs replicating
		case msg := <-leader.msgs:
			fmt.Println("replicating")
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
