package gorp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/rpc"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type Role interface {
	// RPC methods
	RequestVote(RequestVoteMessage, *RequestVoteReply) error
	AppendMessage(AppendMessage, *AppendMessageReply) error

	// Used to implement the actual logic of the replicas/individual roles
	// Returns the next role to transition to
	Execute(context.Context)

	HandleClient(w http.ResponseWriter, r *http.Request)

	GetChangeSignal() chan *RoleTransition

	// Used as a type check to ensure that the role has a relation
	// to the underlying state object of the replica
	GetState() *State
}

// A wrapper for the state/role so that RPC calls don't have to be specific
// about the concrete type it is making the call to. All its methods forward to
// the Role struct it has an instance of.
type Broker struct {
	Role Role

	rpc_server    *http.Server
	client_server *http.Server
}

func (broker *Broker) StartRPCServer() {

	router := mux.NewRouter()

	server := rpc.NewServer()
	server.Register(broker)

	router.Handle("/", server)

	broker.rpc_server = &http.Server{
		Addr:    broker.Role.GetState().Host,
		Handler: router,
	}

	// start the RPC server
	go broker.rpc_server.ListenAndServe()
}

func (broker *Broker) StopRPCServer() {
	broker.rpc_server.Shutdown(context.Background())
}

func (broker *Broker) StartClientServer() {
	rpc_port, err := strconv.Atoi(strings.Split(broker.Role.GetState().Host, ":")[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	port := rpc_port + 3000

	mux := http.NewServeMux()

	mux.HandleFunc("/", broker.Role.HandleClient)

	// Debug routes
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		broker.Role.GetChangeSignal() <- nil
		w.WriteHeader(200)
	})
	mux.HandleFunc("/demote", func(w http.ResponseWriter, r *http.Request) {
		broker.Role.GetChangeSignal() <- &RoleTransition{RoleName: "follower", State: broker.Role.GetState()}
		w.WriteHeader(200)
	})
	mux.HandleFunc("/health-check", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		health := fmt.Sprintf("role: %s, term: %d, commit: %d", broker.Role.GetState().Role, broker.Role.GetState().CommitTerm, broker.Role.GetState().CommitIndex)
		json.NewEncoder(w).Encode(health)
	})

	broker.client_server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go broker.client_server.ListenAndServe()
}

func (broker *Broker) StopClientServer() {
	broker.client_server.Shutdown(context.Background())
}

func (broker *Broker) RequestVote(rvm RequestVoteMessage, rvp *RequestVoteReply) error {
	return broker.Role.RequestVote(rvm, rvp)
}

func (broker *Broker) AppendMessage(ap AppendMessage, apr *AppendMessageReply) error {
	return broker.Role.AppendMessage(ap, apr)
}

func (broker *Broker) Execute(ctx context.Context) {
	broker.Role.Execute(ctx)
}

func (broker *Broker) NextRole(ctx context.Context) (Role, error) {
	select {
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	case transition := <-broker.Role.GetChangeSignal():
		if transition == nil || transition.RoleName == "" {
			return nil, errors.New("stopped")
		}

		var role Role
		switch strings.ToLower(transition.RoleName) {
		case "leader":
			l := &Leader{}
			l.Init(transition.State)
			role = l
		case "candidate":
			c := &Candidate{}
			c.Init(transition.State)
			role = c
		case "follower":
			f := &Follower{}
			f.Init(transition.State)
			role = f
		default:
			return nil, errors.New("unknown role: " + transition.RoleName)
		}
		return role, nil
	}
}

func (broker *Broker) SwitchRole(role Role) {
	broker.Role = role
}

// For start up
func FromState(state *State) Broker {

	var role Role
	switch strings.ToLower(state.Role) {
	case "leader":
		l := &Leader{}
		l.Init(state)
		role = l
	case "candidate":
		c := &Candidate{}
		c.Init(state)
		role = c
	default:
		f := &Follower{}
		f.Init(state)
		role = f
	}

	return Broker{Role: role}
}

// Factory functions to create new role instances
func NewLeader() *Leader {
	return &Leader{}
}

func NewCandidate() *Candidate {
	return &Candidate{}
}

func NewFollower() *Follower {
	return &Follower{}
}
