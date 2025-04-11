package gorp_role

import (
	"context"
	"fmt"
	"net/rpc"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

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
