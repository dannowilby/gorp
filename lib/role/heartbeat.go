package gorp_role

import (
	"context"
	"fmt"
	"net/rpc"
	"time"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
)

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
