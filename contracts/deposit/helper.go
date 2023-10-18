package deposit

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// NewDepositContractCallerFromBoundContract creates a new instance of DepositContractCaller, bound to
// a specific deployed contract.
func NewDepositContractCallerFromBoundContract(contract *bind.BoundContract) ContractCaller {
	return ContractCaller{contract: contract}
}

// NewDepositContractTransactorFromBoundContract creates a new instance of
// DepositContractTransactor, bound to a specific deployed contract.
func NewDepositContractTransactorFromBoundContract(contract *bind.BoundContract) ContractTransactor {
	return ContractTransactor{contract: contract}
}

// NewDepositContractFiltererFromBoundContract creates a new instance of
// DepositContractFilterer, bound to a specific deployed contract.
func NewDepositContractFiltererFromBoundContract(contract *bind.BoundContract) ContractFilterer {
	return ContractFilterer{contract: contract}
}
