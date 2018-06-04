package sharding

import (
	"fmt"
	"math/big"
	"reflect"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// Node defines a a sharding-enabled Ethereum instance that provides
// full control and shared access of necessary components and services
// for a sharded Ethereum blockchain.
type Node interface {
	Start() error
	Close() error
	Register(constructor ServiceConstructor) error
	SMCClient() SMCClient
}

// SMCClient contains useful methods for a sharding node to interact with
// an Ethereum client running on the mainchain.
type SMCClient interface {
	Account() *accounts.Account
	CreateTXOpts(value *big.Int) (*bind.TransactOpts, error)
	SMCCaller() *contracts.SMCCaller
	SMCTransactor() *contracts.SMCTransactor
	ChainReader() ethereum.ChainReader
	DepositFlag() bool
	SetDepositFlag(deposit bool)
	DataDirPath() string
}

// ShardP2P defines an interface for a peer-to-peer service in a
// sharded Ethereum blockchain.
type ShardP2P interface{}

// TXPool defines an interface for a transaction pool service that handles
// incoming shard transactions in the network.
type TXPool interface{}

// Actor refers to either a notary, proposer, or observer.
type Actor interface {
	Start() error
	Stop() error
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

// Service is an individual protocol that can be registered into a node.
type Service interface {
	// Start is called after all services have been constructed to
	// spawn any goroutines required by the service.
	Start() error
	// Stop terminates all goroutines belonging to the service,
	// blocking until they are all terminated.
	Stop() error
}

// Service retrieves a currently running service registered of a specific type.
func (ctx *ServiceContext) Service(service interface{}) error {
	element := reflect.ValueOf(service).Elem()
	if running, ok := ctx.Services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("unknown service")
}
