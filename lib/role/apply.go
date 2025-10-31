package gorp_role

import "fmt"

// Applies the log entries up to the commit index, updating the `LastApplied` to
// reflect this.
func Apply(role Role) {
	log := role.GetState().Log
	up_to := role.GetState().CommitIndex
	last_applied := role.GetState().LastApplied

	for last_applied != up_to {
		entry := log[last_applied+1]

		if entry.Type == "data" {
			fmt.Println("applying:", last_applied+1)
		}
		if entry.Type == "config" {
			fmt.Println("updating config")

			// requeue a c_new message if this machine is the leader
		}

		last_applied++
		role.GetState().LastApplied = last_applied
	}
}
