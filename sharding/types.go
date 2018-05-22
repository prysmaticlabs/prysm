package sharding

import (
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// Service defines items that can be registered into a sharding client.
//
// life-cycle management is delegated to the sharding client. The service is allowed to
// initialize itself upon creation, but no goroutines should be spun up outside of the
// Start method.
type Service interface {
	// Start is called after all services have been constructed and the networking
	// layer was also initialized to spawn any goroutines required by the service.
	Start() error

	// Stop terminates all goroutines belonging to the service, blocking until they
	// are all terminated.
	Stop() error
}

// Node methods that must be implemented to run a sharding node.
type Node interface {
	Start() error
	Close()
	CreateTXOpts(*big.Int) (*bind.TransactOpts, error)
	ChainReader() ethereum.ChainReader
	Account() *accounts.Account
	SMCCaller() *contracts.SMCCaller
	SMCTransactor() *contracts.SMCTransactor
	DepositFlagSet() bool
}
