
# gorp
[![Go](https://github.com/dannowilby/gorp/actions/workflows/go.yml/badge.svg)](https://github.com/dannowilby/gorp/actions/workflows/go.yml)

A distributed photo storage implemented with Raft [1].

## Quick start
To run a quick cluster with 3 instances, run the following command. Make sure
your `config.local.json` is correct.

`go run . -id=<i> -config=config.local.json`

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
- [ ] **Log compaction** - snapshots of the machine's log need to be stored on
  disk to save space and allow better recoverability/resiliency from crashes

## Design considerations
Embarrasingly a Kubernetes resource was designed at first. After more research,
this was determined to be a ouroboros-like idea. Kubernetes uses etcd to ensure
consistency between nodes/pods in a cluster. etcd is literally an implementation
of Raft, so relying on it does not make the most amount of sense.

## References

[1] [In Search of an Understandable Consensus Algorithm (Extended Version)](https://raft.github.io/raft.pdf), Diego Ongaro and John Ousterhout
