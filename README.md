
# gorp
[![Go](https://github.com/dannowilby/gorp/actions/workflows/go.yml/badge.svg)](https://github.com/dannowilby/gorp/actions/workflows/go.yml)

A distributed photo store implemented with Raft [1]. Read about the development [here]().

![gorp-ui preview](./preview.mp4)

## Getting started
1. Start your replicas, for example:
   `go run . -config config.local.json`
2. Serve the frontend directory:
   `cd frontend && python3 -m http.server 5173`
3. Open `http://localhost:5173`.
4. Upload your cluster config JSON in the UI before sending operations.

Frontend pages:
- `frontend/index.html`: photo operations (upload/list/download/update/delete + raw operations).
- `frontend/config.html`: upload a new config and broadcast `type: "config"` messages.


## Implemented features

### Raft
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

### Photo store frontend
- [x] **File uploads** - 
- [x] **File downloads** - 

## Disclaimer

## References

[1] [In Search of an Understandable Consensus Algorithm (Extended Version)](https://raft.github.io/raft.pdf), Diego Ongaro and John Ousterhout
