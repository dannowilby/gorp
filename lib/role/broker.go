package gorp_role

import (
	"context"
	"net/http"
	"net/rpc"
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

	NextRole(context.Context) (Role, error)

	// Used to implement the actual logic of the replicas/individual roles
	// Returns the next role to transition to
	Execute(context.Context)

	// Used as a type check to ensure that the role has a relation
	// to the underlying state object of the replica
	GetState() *gorp.State
}

// Get the number of machines needed for a majority, does remove the calling
// machine from the majority. AKA the number of separate machines needed for a majority.
func NumMajority(role Role) int {
	return ((len(role.GetState().Config) + 1) / 2) - 1
}

// A wrapper for the state/role so that RPC calls don't have to be specific
// about the concrete type it is making the call to. All its methods forward to
// the Role struct it has an instance of.
type Broker struct {
	Role Role

	server *http.Server
}

func (broker *Broker) StartServer() {

	router := mux.NewRouter()

	server := rpc.NewServer()
	server.Register(broker)

	router.Handle("/", server)

	broker.server = &http.Server{
		Addr:    broker.Role.GetState().Host,
		Handler: router,
	}

	// start the RPC server
	go broker.server.ListenAndServe()
}

func (broker *Broker) StopServer() {
	broker.server.Shutdown(context.Background())
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
	return broker.Role.NextRole(ctx)
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
