package gorp_role

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/rpc"
	"strconv"
	"strings"

	gorp "github.com/dannowilby/gorp/lib"
	gorp_rpc "github.com/dannowilby/gorp/lib/rpc"
	"github.com/gorilla/mux"
)

type Role interface {
	Init(*gorp.State) Role

	// RPC methods
	RequestVote(gorp_rpc.RequestVoteMessage, *gorp_rpc.RequestVoteReply) error
	AppendMessage(gorp_rpc.AppendMessage, *gorp_rpc.AppendMessageReply) error

	// Used to implement the actual logic of the replicas/individual roles
	// Returns the next role to transition to
	Execute(context.Context)

	HandleClient(w http.ResponseWriter, r *http.Request)

	GetChangeSignal() chan Role

	// Used as a type check to ensure that the role has a relation
	// to the underlying state object of the replica
	GetState() *gorp.State
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
		broker.Role.GetChangeSignal() <- new(Follower).Init(broker.Role.GetState())
		w.WriteHeader(200)
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

func (broker *Broker) RequestVote(rvm gorp_rpc.RequestVoteMessage, rvp *gorp_rpc.RequestVoteReply) error {
	return broker.Role.RequestVote(rvm, rvp)
}

func (broker *Broker) AppendMessage(ap gorp_rpc.AppendMessage, apr *gorp_rpc.AppendMessageReply) error {
	return broker.Role.AppendMessage(ap, apr)
}

func (broker *Broker) Execute(ctx context.Context) {
	broker.Role.Execute(ctx)
}

func (broker *Broker) NextRole(ctx context.Context) (Role, error) {
	select {
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	case next_role := <-broker.Role.GetChangeSignal():
		if next_role != nil {
			return next_role, nil
		}
		return nil, errors.New("stopped")
	}
}

func (broker *Broker) SwitchRole(role Role) {
	broker.Role = role
}

// For start up
func FromState(state *gorp.State) Broker {

	var role Role
	switch strings.ToLower(state.Role) {
	case "leader":
		role = new(Leader).Init(state)
	case "candidate":
		role = new(Candidate).Init(state)
	default:
		role = new(Follower).Init(state)
	}

	return Broker{Role: role}
}
