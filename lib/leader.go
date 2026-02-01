package gorp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Leader struct {
	State *State

	// essentially a queue of messages that we have to try and append
	MessageQueue chan LogEntry

	// the next log index to send to the associated server
	nextIndex map[string]int

	// index of highest log entry known to be replicated on the server
	matchIndex map[string]int

	ChangeSignal chan *RoleTransition
}

func (leader *Leader) Init(state *State) *Leader {
	leader.State = state
	leader.State.Role = "leader"

	leader.MessageQueue = make(chan LogEntry, 10)
	leader.ChangeSignal = make(chan *RoleTransition, 1)

	leader.nextIndex = make(map[string]int)
	leader.matchIndex = make(map[string]int)

	for _, server := range state.Config {
		leader.nextIndex[server] = len(leader.State.Log)
		leader.matchIndex[server] = 0
	}

	fmt.Println("Leader:", leader.State.Host)

	return leader
}

func (leader *Leader) RequestVote(msg RequestVoteMessage, rply *RequestVoteReply) error {
	if VoteMsgIsUpToDate(leader.State, &msg) {
		rply.VoteGranted = true
		rply.Term = leader.State.CommitTerm
		return nil
	}

	rply.VoteGranted = false
	rply.Term = leader.State.CommitTerm

	return nil
}

func (leader *Leader) AppendMessage(msg AppendMessage, rply *AppendMessageReply) error {
	// we need to check if the length of the logs is longer and if the term is
	// the same, if true, then
	if !AppendMessageIsNewer(leader.State, &msg) {

		// the msg sender has a longer log, therefore it should be the leader
		if AppendMessageLogIsLonger(leader.State, &msg) {
			leader.ChangeSignal <- &RoleTransition{RoleName: "follower", State: leader.State}
		}

		rply.CommitTerm = leader.State.CommitTerm
		rply.Success = false
		return nil
	}

	// unlike the follower, we don't modify anything else
	// this allows all the behavior that handles log synchronization by the
	// follower role

	rply.CommitTerm = leader.State.CommitTerm
	rply.Success = true

	leader.ChangeSignal <- &RoleTransition{RoleName: "follower", State: leader.State}

	return nil
}

func (leader *Leader) GetStatus(hash string) StatusResponse {
	electionTimeout := time.Duration(leader.State.ElectionTimeout) * time.Millisecond

	// Walk the committed portion of the log looking for the hash
	for i := 0; i <= leader.State.CommitIndex; i++ {
		if leader.State.Log[i].Hash == hash {
			return StatusResponse{
				Status: "success",
				Hash:   hash,
			}
		}
	}

	// Not committed yet, check if it's in the uncommitted portion
	for i := leader.State.CommitIndex + 1; i < len(leader.State.Log); i++ {
		entry := leader.State.Log[i]
		if entry.Hash == hash {
			// Found but not committed, check if we're still within the timeout
			if time.Since(entry.Timestamp) < electionTimeout {
				return StatusResponse{
					Status: "pending",
					Hash:   hash,
				}
			}

			// Past the election timeout and still not committed
			return StatusResponse{
				Status: "failed",
				Hash:   hash,
			}
		}
	}

	// Hash not found anywhere in the log
	return StatusResponse{
		Status: "failed",
		Hash:   hash,
	}
}

// HandleClient routes incoming client requests to the appropriate handler.
//
// Endpoints:
//
//	POST /api
//	  Submit a new log entry (write, update, delete) to the cluster.
//	  Request body:
//	    {
//	      "type": "data",
//	      "message": {
//	        "path": "documents/hello.txt",
//	        "blob": "hello world",
//	        "operation": "write"
//	      }
//	    }
//	  Response (200):
//	    {
//	      "hash": "abc123...",
//	      "timestamp": "2024-01-01T00:00:00.000000000Z"
//	    }
//
//	  Examples:
//
//	    Write a file:
//	      curl -X POST http://localhost:8080/api \
//	        -H "Content-Type: application/json" \
//	        -d '{"type":"data","message":{"path":"docs/hello.txt","blob":"hello world","operation":"write"}}'
//
//	    Update a file:
//	      curl -X POST http://localhost:8080/api \
//	        -H "Content-Type: application/json" \
//	        -d '{"type":"data","message":{"path":"docs/hello.txt","blob":"updated content","operation":"update"}}'
//
//	    Delete a file:
//	      curl -X POST http://localhost:8080/api \
//	        -H "Content-Type: application/json" \
//	        -d '{"type":"data","message":{"path":"docs/hello.txt","blob":"","operation":"delete"}}'
//
//	GET /status?hash=<hash>
//	  Poll for the status of a previously submitted log entry.
//	  Response (200):
//	    { "status": "success", "hash": "abc123..." }
//	    { "status": "pending", "hash": "abc123..." }
//	    { "status": "failed",  "hash": "abc123..." }
//
//	  Example:
//	    curl "http://localhost:8080/status?hash=abc123..."
//
//	GET /file?path=<path>
//	  Read a file from the committed state machine.
//	  Response (200): "file contents here"
//	  Response (404): "File not found."
//
//	  Example:
//	    curl "http://localhost:8080/file?path=docs/hello.txt"
//
//	GET /files
//	  List all files in the data directory.
//	  Response (200):
//	    ["docs/hello.txt", "docs/world.txt"]
func (leader *Leader) HandleClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/status":
		leader.handleStatus(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/file":
		leader.handleFileRead(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/files":
		leader.handleFileList(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api":
		leader.handleSubmit(w, r)
	default:
		w.WriteHeader(404)
		json.NewEncoder(w).Encode("Not found.")
	}
}

func (leader *Leader) handleFileList(w http.ResponseWriter, r *http.Request) {
	var files []string

	err := filepath.Walk("data", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Strip the "data/" prefix so paths are relative
			relative, err := filepath.Rel("data", path)
			if err != nil {
				return err
			}
			files = append(files, relative)
		}
		return nil
	})

	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode("Error listing files.")
		return
	}

	// Return an empty array instead of null if no files exist
	if files == nil {
		files = []string{}
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(files)
}

func (leader *Leader) handleFileRead(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode("Missing 'path' query parameter.")
		return
	}

	data, err := os.ReadFile(filepath.Join("data", path))
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode("File not found.")
		} else {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode("Error reading file.")
		}
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(string(data))
}

func (leader *Leader) handleStatus(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode("Missing 'hash' query parameter.")
		return
	}

	status := leader.GetStatus(hash)
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(status)
}

func (leader *Leader) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var message LogEntry
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode("Invalid request body.")
		return
	}
	message.Term = leader.State.CommitTerm
	message.Timestamp = time.Now()

	hash, err := GenerateHash(message.Message)
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode("Failed to generate message hash.")
		return
	}
	message.Hash = hash

	if message.Type == "config" {
		var config ConfigData
		err := json.Unmarshal(message.Message, &config)
		if err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode("Invalid configuration format.")
			return
		}
		config.Old = leader.State.Config
	}

	leader.MessageQueue <- message

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(SubmitResponse{
		Hash:      hash,
		Timestamp: message.Timestamp.Format(time.RFC3339Nano),
	})
}

func (leader *Leader) Execute(ctx context.Context) {

	// start continuously heartbeating
	go leader.SendHeartbeats(ctx)

	for {
		select {
		// when we have a message that needs replicating
		case msg := <-leader.MessageQueue:
			leader.Replicate(ctx, msg)
		case <-ctx.Done():
			return
		}
	}

}

func (leader *Leader) GetChangeSignal() chan *RoleTransition {
	return leader.ChangeSignal
}

func (leader *Leader) GetState() *State {
	return leader.State
}
