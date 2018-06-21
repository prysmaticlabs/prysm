package sharding

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

// Node defines a a sharding-enabled Ethereum instance that provides
// full control and shared access of necessary components and services
// for a sharded Ethereum blockchain.
type Node interface {
	Start()
	Close()
	Register(constructor ServiceConstructor) error
}

// Actor refers to either a notary, proposer, or observer in the sharding spec.
type Actor interface {
	Service
	// TODO: will actors have actor-specific methods? To be decided.
}

// CollationFetcher defines functionality for a struct that is able to extract
// respond with collation information to the caller. Shard implements this interface.
type CollationFetcher interface {
	CollationByHeaderHash(headerHash *common.Hash) (*Collation, error)
}

// ServiceContext is a collection of service independent options inherited from
// the protocol stack, that is passed to all constructors to be optionally used;
// as well as utility methods to operate on the service environment.
type ServiceContext struct {
	Services map[reflect.Type]Service // Index of the already constructed services
}

// ServiceConstructor is the function signature of the constructors needed to be
// registered for service instantiation.
type ServiceConstructor func(ctx *ServiceContext) (Service, error)

// Service is an individual protocol that can be registered into a node. Having a sharding
// node maintain a service registry allows for easy, shared-dependencies. For example,
// a proposer service might depend on a p2p server, a txpool, an smc client, etc.
type Service interface {
	// Start is called after all services have been constructed to
	// spawn any goroutines required by the service.
	Start()
	// Stop terminates all goroutines belonging to the service,
	// blocking until they are all terminated.
	Stop() error
}

// RetrieveService sets the `service` argument to a currently running service
// registered of a specific type.
func (ctx *ServiceContext) RetrieveService(service interface{}) error {
	element := reflect.ValueOf(service).Elem()
	if running, ok := ctx.Services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("unknown service: %T", service)
}
