package gorp_role

// Applies the log entries up to the commit index, updating the `LastApplied` to
// reflect this.
func Apply(role Role) {
	log := role.GetState().Log
	up_to := role.GetState().CommitIndex
	last_applied := role.GetState().LastApplied

	for last_applied != up_to {
		entry := log[last_applied+1]

		if entry.Type == "data" {
			// fmt.Println("applying:", last_applied+1)
		}
		if entry.Type == "config" {
			// fmt.Println("updating config")
		}

		last_applied++
		role.GetState().LastApplied = last_applied
	}
}
