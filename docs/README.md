
# gorp

A distributed photo app implemented with Raft.

## Implemented features

- [x] Message appending
- [ ] Config changes
- [ ] Leadership consensus
- [ ] Log replication
- [ ] Log compaction

## Implementation/prototyping details

Currently the implementation uses the RPC package from the standard library.
This causes some issues with the current architecture. When creating an RPC to
another instance, it is expected that the instance making the call specifies the
object that the call is acting on. 

For instance, making a RequestVote RPC to a follower looks like this
`client.Call("Follower.RequestVote", &msg, &rply)`. It is evident this causes
issues when we don't know what the target instance is.

The solution (theorized so far) is to use a more abstract struct to host the RPC calls, calling on
the implementations of each instance role through some reification of the state
pattern. The details of which are slowly coming more into focus.