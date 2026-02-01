package gorp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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

func (follower *Follower) Apply() {
	log := follower.State.Log
	up_to := follower.State.CommitIndex
	last_applied := follower.State.LastApplied

	for last_applied != up_to {
		entry := log[last_applied+1]

		if entry.Type == "data" {
			fmt.Println("applying:", last_applied+1)
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

			// update the config
			follower.State.Config = append(config.New, config.Old...)
		}

		last_applied++
		follower.State.LastApplied = last_applied
	}
}
