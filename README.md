# gorp
[![Go](https://github.com/dannowilby/gorp/actions/workflows/go.yml/badge.svg)](https://github.com/dannowilby/gorp/actions/workflows/go.yml)

A Raft implementation [1]. Before I pivoted/focused the direction of this
project, I wrote about the why, what, and how
[here](https://medium.com/@dywilby/building-something-for-someone-you-love-b850111fb27a)
and created video demo of it
[here](https://www.youtube.com/watch?v=SxVlT8gQOqI).

## Getting started
These docs are still a heavy work in progress, so any ambiguities or depreciations will be updated shortly.

### Starting a gorp cluster
`config.local.json` is an example configuration file. The configuration file can
be supplied to an instance on its startup by setting the `-config` flag. Specify
all the members of the cluster in the configuration file.

Another way to start a cluster is to individually execute the binary, and then
send a configuration change request with your desired cluster members.

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

## The future of this project
The project is being pivoted! The photo store and raft protocol implementation
is being decoupled. This repo will just be the protocol, and more effort is
being put into the following areas:
1. Logging - adding debug information for appending messages, elections, etc...
2. Snapshotting - compressing and clearing the in-memory log
3. Comparisons to existing implementations - time to detect and recover a
   crashed leader (comparing to the raft paper), queries per second (comparing
   to etcd), etc...
4. Full-system testing - running clusters under particular conditions and
   automatically verifying correct behavior

## References

[1] [In Search of an Understandable Consensus Algorithm (Extended
Version)](https://raft.github.io/raft.pdf), Diego Ongaro and John Ousterhout
