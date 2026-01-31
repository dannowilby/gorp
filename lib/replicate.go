package gorp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/rpc"
	"os"
	"path/filepath"
)

// Updates the machine and tries to append
func (leader *Leader) updateMachine(host string, success chan bool) {

	client, err := rpc.DialHTTPPath("tcp", host, "/")

	if err != nil {
		fmt.Println(err)
		success <- false
		return
	}

	log_len := len(leader.State.Log)
	msg_to_send := leader.nextIndex[host]

	for msg_to_send != log_len {

		// something has gone wrong, maybe the machine is unreachable
		if msg_to_send == -1 {
			leader.nextIndex[host] = 0
			success <- false
			return
		}

		prev_index := msg_to_send - 1
		prev_term := -1
		if prev_index > -1 {
			prev_term = leader.State.Log[prev_index].Term
		}

		append_message_args := AppendMessage{
			Term:     leader.State.CommitTerm,
			LeaderId: leader.State.Host,

			PrevLogTerm:  prev_term,
			PrevLogIndex: prev_index,

			Entry: leader.State.Log[msg_to_send],

			LeaderCommit: leader.State.CommitIndex,
		}

		append_message_rply := AppendMessageReply{}
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
	}

	success <- true
}

// In its current form, this implementation may cause issues, cancelling
// normally functioning appends when a majority is achieved. This needs further
// testing, but it should still offer certainty that a majority has replicated
// any one log message.
func (leader *Leader) Replicate(parent_ctx context.Context, msg LogEntry) {

	ctx, cancel := context.WithCancel(parent_ctx)

	leader.State.Log = append(leader.State.Log, msg)

	// query machines
	accepted := make(chan bool)
	for _, host := range leader.State.Config {
		if host == leader.State.Host {
			continue
		}

		// send message to machines, get response through accepted channel
		go leader.updateMachine(host, accepted)
	}

	majority := NumMajority(leader.State)
	vote_count := 0

	for {
		select {
		case success := <-accepted:

			if success {
				vote_count++
			}

			if vote_count >= majority {

				// commit the log
				leader.State.CommitIndex += 1

				// apply the log
				leader.Apply()

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

func (leader *Leader) Apply() {
	log := leader.State.Log
	up_to := leader.State.CommitIndex
	last_applied := leader.State.LastApplied

	for last_applied != up_to {
		entry := log[last_applied+1]

		if entry.Type == "data" {
			fmt.Println("Applying:", last_applied+1)
			ApplyData(entry)
		}
		if entry.Type == "config" {
			fmt.Println("Updating config.")

			var config ConfigData

			// should not error due to previous error checking
			err := json.Unmarshal(entry.Message, &config)

			if err != nil {
				fmt.Println(err)
			}

			if len(config.Old) > 0 {

				// build the new data
				data, _ := json.Marshal(ConfigData{New: config.New})

				// requeue the new data
				leader.MessageQueue <- LogEntry{
					Term:    leader.State.CommitTerm,
					Type:    "config",
					Message: data,
				}
			}

			// update the config
			leader.State.Config = append(config.New, config.Old...)
		}

		last_applied++
		leader.State.LastApplied = last_applied
	}
}

const dataDir = "data"

func ApplyData(entry LogEntry) error {

	fmt.Println("Applying data entry")
	var message MessageData
	err := json.Unmarshal(entry.Message, &message)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	fullPath := filepath.Join(dataDir, message.Path)

	// Ensure the data directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	switch message.Operation {
	case "write":
		fmt.Println("writing")
		// Create or overwrite a file
		if err := os.WriteFile(fullPath, []byte(message.Blob), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Println("Wrote file:", fullPath)

	case "update":
		// Only update if the file already exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("file does not exist, cannot update: %s", fullPath)
		}
		if err := os.WriteFile(fullPath, []byte(message.Blob), 0644); err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}
		fmt.Println("Updated file:", fullPath)

	case "delete":
		// Remove the file if it exists
		if err := os.Remove(fullPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist, cannot delete: %s", fullPath)
			}
			return fmt.Errorf("failed to delete file: %w", err)
		}
		fmt.Println("Deleted file:", fullPath)

	default:
		return fmt.Errorf("unknown operation: %s", message.Operation)
	}

	return nil
}
