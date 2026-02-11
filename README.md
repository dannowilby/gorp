
# gorp
[![Go](https://github.com/dannowilby/gorp/actions/workflows/go.yml/badge.svg)](https://github.com/dannowilby/gorp/actions/workflows/go.yml)

A distributed photo store implemented with Raft [1]. Read about the development [here]().

## Getting started

### Starting a gorp instance
Download or build the executable, and then run it. You can either set the
`config` flag to point at a cluster configuration file, or you can manually send
the file to the config endpoint. If using the config flag, make sure to set the
`id` flag in conjunction to let the instance know which one it is.

### Starting the frontend
Change the `API_BASE` variable to point at one of the instances in your cluster.
Then run the project normally.

## Implemented features
- [x] **Message appending** - follower replicas need to be able to add new
  messages to the end of their log when the appropriate conditions are met
  (message term is valid, log is up-to-date, the previous message of both
  replicas match).
- [x] **Leadership consensus** - if a system is disturbed in some way, like a
  network partition or leader crash, then the system needs to recover in a
  timely way. This is done through an election.
- [x] **Log replication** - the follower logs needs to heal themselves to be
  up-to-date from the leader's logs. This is done through an inductive process.
- [x] **Config changes** - special messages can be passed that define updated
  network configurations. The new set of hosts in the configuration will be
  transitioned to while perserving the protocol invariants.
- [x] **Status tracking** - Raft sacrificies availability over consistency and
  partionability; this means that a message's status needs to be tracked by the
  user for success, in this project's case using unique hashes
- [x] **User file operations** - the frontend allows automatic tracking of upload, update,
  and delete operations

## References

[1] [In Search of an Understandable Consensus Algorithm (Extended Version)](https://raft.github.io/raft.pdf), Diego Ongaro and John Ousterhout
