// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package vrc

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// ValidatorRegistrationABI is the input ABI used to generate the binding from.
const ValidatorRegistrationABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"VALIDATOR_DEPOSIT\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"usedHashedPubkey\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_pubkey\",\"type\":\"bytes\"},{\"name\":\"_withdrawalShardID\",\"type\":\"uint256\"},{\"name\":\"_withdrawalAddressbytes32\",\"type\":\"address\"},{\"name\":\"_randaoCommitment\",\"type\":\"bytes32\"}],\"name\":\"deposit\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"hashedPubkey\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"withdrawalShardID\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"withdrawalAddressbytes32\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"randaoCommitment\",\"type\":\"bytes32\"}],\"name\":\"ValidatorRegistered\",\"type\":\"event\"}]"

// ValidatorRegistrationBin is the compiled bytecode used for deploying new contracts.
const ValidatorRegistrationBin = `0x608060405234801561001057600080fd5b50610406806100206000396000f3006080604052600436106100565763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663441d92cc811461005b5780638618d77814610082578063878df5ac146100ae575b600080fd5b34801561006757600080fd5b5061007061011e565b60408051918252519081900360200190f35b34801561008e57600080fd5b5061009a60043561012b565b604080519115158252519081900360200190f35b6040805160206004803580820135601f810184900484028501840190955284845261011c9436949293602493928401919081908401838280828437509497505084359550505050602082013573ffffffffffffffffffffffffffffffffffffffff1691604001359050610140565b005b6801bc16d674ec80000081565b60006020819052908152604090205460ff1681565b6000346801bc16d674ec800000146101b957604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601b60248201527f496e636f72726563742076616c696461746f72206465706f7369740000000000604482015290519081900360640190fd5b845160301461022957604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601a60248201527f5075626c6963206b6579206973206e6f74203438206279746573000000000000604482015290519081900360640190fd5b846040516020018082805190602001908083835b6020831061025c5780518252601f19909201916020918201910161023d565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106102bf5780518252601f1990920191602091820191016102a0565b51815160209384036101000a60001901801990921691161790526040805192909401829003909120600081815291829052929020549194505060ff1615915061036b905057604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f5075626c6963206b657920616c72656164792075736564000000000000000000604482015290519081900360640190fd5b60008181526020818152604091829020805460ff1916600117905581518681529151849273ffffffffffffffffffffffffffffffffffffffff87169285927f7b0678aab009b61a805f5004869728b53a444f9a3e6bb9e22b8537c89af512749281900390910190a450505050505600a165627a7a72305820f94ed1bea88aec62badbde73ec4adcd500e067fa0245052ea42e662c529b94870029`

// DeployValidatorRegistration deploys a new Ethereum contract, binding an instance of ValidatorRegistration to it.
func DeployValidatorRegistration(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ValidatorRegistration, error) {
	parsed, err := abi.JSON(strings.NewReader(ValidatorRegistrationABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ValidatorRegistrationBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ValidatorRegistration{ValidatorRegistrationCaller: ValidatorRegistrationCaller{contract: contract}, ValidatorRegistrationTransactor: ValidatorRegistrationTransactor{contract: contract}, ValidatorRegistrationFilterer: ValidatorRegistrationFilterer{contract: contract}}, nil
}

// ValidatorRegistration is an auto generated Go binding around an Ethereum contract.
type ValidatorRegistration struct {
	ValidatorRegistrationCaller     // Read-only binding to the contract
	ValidatorRegistrationTransactor // Write-only binding to the contract
	ValidatorRegistrationFilterer   // Log filterer for contract events
}

// ValidatorRegistrationCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValidatorRegistrationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorRegistrationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValidatorRegistrationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorRegistrationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValidatorRegistrationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorRegistrationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValidatorRegistrationSession struct {
	Contract     *ValidatorRegistration // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// ValidatorRegistrationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValidatorRegistrationCallerSession struct {
	Contract *ValidatorRegistrationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// ValidatorRegistrationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValidatorRegistrationTransactorSession struct {
	Contract     *ValidatorRegistrationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// ValidatorRegistrationRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValidatorRegistrationRaw struct {
	Contract *ValidatorRegistration // Generic contract binding to access the raw methods on
}

// ValidatorRegistrationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValidatorRegistrationCallerRaw struct {
	Contract *ValidatorRegistrationCaller // Generic read-only contract binding to access the raw methods on
}

// ValidatorRegistrationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValidatorRegistrationTransactorRaw struct {
	Contract *ValidatorRegistrationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValidatorRegistration creates a new instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistration(address common.Address, backend bind.ContractBackend) (*ValidatorRegistration, error) {
	contract, err := bindValidatorRegistration(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistration{ValidatorRegistrationCaller: ValidatorRegistrationCaller{contract: contract}, ValidatorRegistrationTransactor: ValidatorRegistrationTransactor{contract: contract}, ValidatorRegistrationFilterer: ValidatorRegistrationFilterer{contract: contract}}, nil
}

// NewValidatorRegistrationCaller creates a new read-only instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistrationCaller(address common.Address, caller bind.ContractCaller) (*ValidatorRegistrationCaller, error) {
	contract, err := bindValidatorRegistration(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationCaller{contract: contract}, nil
}

// NewValidatorRegistrationTransactor creates a new write-only instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistrationTransactor(address common.Address, transactor bind.ContractTransactor) (*ValidatorRegistrationTransactor, error) {
	contract, err := bindValidatorRegistration(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationTransactor{contract: contract}, nil
}

// NewValidatorRegistrationFilterer creates a new log filterer instance of ValidatorRegistration, bound to a specific deployed contract.
func NewValidatorRegistrationFilterer(address common.Address, filterer bind.ContractFilterer) (*ValidatorRegistrationFilterer, error) {
	contract, err := bindValidatorRegistration(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationFilterer{contract: contract}, nil
}

// bindValidatorRegistration binds a generic wrapper to an already deployed contract.
func bindValidatorRegistration(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ValidatorRegistrationABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidatorRegistration *ValidatorRegistrationRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ValidatorRegistration.Contract.ValidatorRegistrationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidatorRegistration *ValidatorRegistrationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.ValidatorRegistrationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidatorRegistration *ValidatorRegistrationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.ValidatorRegistrationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidatorRegistration *ValidatorRegistrationCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ValidatorRegistration.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidatorRegistration *ValidatorRegistrationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidatorRegistration *ValidatorRegistrationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.contract.Transact(opts, method, params...)
}

// VALIDATORDEPOSIT is a free data retrieval call binding the contract method 0x441d92cc.
//
// Solidity: function VALIDATOR_DEPOSIT() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) VALIDATORDEPOSIT(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "VALIDATOR_DEPOSIT")
	return *ret0, err
}

// VALIDATORDEPOSIT is a free data retrieval call binding the contract method 0x441d92cc.
//
// Solidity: function VALIDATOR_DEPOSIT() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) VALIDATORDEPOSIT() (*big.Int, error) {
	return _ValidatorRegistration.Contract.VALIDATORDEPOSIT(&_ValidatorRegistration.CallOpts)
}

// VALIDATORDEPOSIT is a free data retrieval call binding the contract method 0x441d92cc.
//
// Solidity: function VALIDATOR_DEPOSIT() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) VALIDATORDEPOSIT() (*big.Int, error) {
	return _ValidatorRegistration.Contract.VALIDATORDEPOSIT(&_ValidatorRegistration.CallOpts)
}

// UsedHashedPubkey is a free data retrieval call binding the contract method 0x8618d778.
//
// Solidity: function usedHashedPubkey( bytes32) constant returns(bool)
func (_ValidatorRegistration *ValidatorRegistrationCaller) UsedHashedPubkey(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "usedHashedPubkey", arg0)
	return *ret0, err
}

// UsedHashedPubkey is a free data retrieval call binding the contract method 0x8618d778.
//
// Solidity: function usedHashedPubkey( bytes32) constant returns(bool)
func (_ValidatorRegistration *ValidatorRegistrationSession) UsedHashedPubkey(arg0 [32]byte) (bool, error) {
	return _ValidatorRegistration.Contract.UsedHashedPubkey(&_ValidatorRegistration.CallOpts, arg0)
}

// UsedHashedPubkey is a free data retrieval call binding the contract method 0x8618d778.
//
// Solidity: function usedHashedPubkey( bytes32) constant returns(bool)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) UsedHashedPubkey(arg0 [32]byte) (bool, error) {
	return _ValidatorRegistration.Contract.UsedHashedPubkey(&_ValidatorRegistration.CallOpts, arg0)
}

// Deposit is a paid mutator transaction binding the contract method 0x878df5ac.
//
// Solidity: function deposit(_pubkey bytes, _withdrawalShardID uint256, _withdrawalAddressbytes32 address, _randaoCommitment bytes32) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactor) Deposit(opts *bind.TransactOpts, _pubkey []byte, _withdrawalShardID *big.Int, _withdrawalAddressbytes32 common.Address, _randaoCommitment [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.contract.Transact(opts, "deposit", _pubkey, _withdrawalShardID, _withdrawalAddressbytes32, _randaoCommitment)
}

// Deposit is a paid mutator transaction binding the contract method 0x878df5ac.
//
// Solidity: function deposit(_pubkey bytes, _withdrawalShardID uint256, _withdrawalAddressbytes32 address, _randaoCommitment bytes32) returns()
func (_ValidatorRegistration *ValidatorRegistrationSession) Deposit(_pubkey []byte, _withdrawalShardID *big.Int, _withdrawalAddressbytes32 common.Address, _randaoCommitment [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.Deposit(&_ValidatorRegistration.TransactOpts, _pubkey, _withdrawalShardID, _withdrawalAddressbytes32, _randaoCommitment)
}

// Deposit is a paid mutator transaction binding the contract method 0x878df5ac.
//
// Solidity: function deposit(_pubkey bytes, _withdrawalShardID uint256, _withdrawalAddressbytes32 address, _randaoCommitment bytes32) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactorSession) Deposit(_pubkey []byte, _withdrawalShardID *big.Int, _withdrawalAddressbytes32 common.Address, _randaoCommitment [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.Deposit(&_ValidatorRegistration.TransactOpts, _pubkey, _withdrawalShardID, _withdrawalAddressbytes32, _randaoCommitment)
}

// ValidatorRegistrationValidatorRegisteredIterator is returned from FilterValidatorRegistered and is used to iterate over the raw logs and unpacked data for ValidatorRegistered events raised by the ValidatorRegistration contract.
type ValidatorRegistrationValidatorRegisteredIterator struct {
	Event *ValidatorRegistrationValidatorRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidatorRegistrationValidatorRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorRegistrationValidatorRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidatorRegistrationValidatorRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidatorRegistrationValidatorRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorRegistrationValidatorRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorRegistrationValidatorRegistered represents a ValidatorRegistered event raised by the ValidatorRegistration contract.
type ValidatorRegistrationValidatorRegistered struct {
	HashedPubkey             [32]byte
	WithdrawalShardID        *big.Int
	WithdrawalAddressbytes32 common.Address
	RandaoCommitment         [32]byte
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterValidatorRegistered is a free log retrieval operation binding the contract event 0x7b0678aab009b61a805f5004869728b53a444f9a3e6bb9e22b8537c89af51274.
//
// Solidity: event ValidatorRegistered(hashedPubkey indexed bytes32, withdrawalShardID uint256, withdrawalAddressbytes32 indexed address, randaoCommitment indexed bytes32)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) FilterValidatorRegistered(opts *bind.FilterOpts, hashedPubkey [][32]byte, withdrawalAddressbytes32 []common.Address, randaoCommitment [][32]byte) (*ValidatorRegistrationValidatorRegisteredIterator, error) {

	var hashedPubkeyRule []interface{}
	for _, hashedPubkeyItem := range hashedPubkey {
		hashedPubkeyRule = append(hashedPubkeyRule, hashedPubkeyItem)
	}

	var withdrawalAddressbytes32Rule []interface{}
	for _, withdrawalAddressbytes32Item := range withdrawalAddressbytes32 {
		withdrawalAddressbytes32Rule = append(withdrawalAddressbytes32Rule, withdrawalAddressbytes32Item)
	}
	var randaoCommitmentRule []interface{}
	for _, randaoCommitmentItem := range randaoCommitment {
		randaoCommitmentRule = append(randaoCommitmentRule, randaoCommitmentItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.FilterLogs(opts, "ValidatorRegistered", hashedPubkeyRule, withdrawalAddressbytes32Rule, randaoCommitmentRule)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationValidatorRegisteredIterator{contract: _ValidatorRegistration.contract, event: "ValidatorRegistered", logs: logs, sub: sub}, nil
}

// WatchValidatorRegistered is a free log subscription operation binding the contract event 0x7b0678aab009b61a805f5004869728b53a444f9a3e6bb9e22b8537c89af51274.
//
// Solidity: event ValidatorRegistered(hashedPubkey indexed bytes32, withdrawalShardID uint256, withdrawalAddressbytes32 indexed address, randaoCommitment indexed bytes32)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) WatchValidatorRegistered(opts *bind.WatchOpts, sink chan<- *ValidatorRegistrationValidatorRegistered, hashedPubkey [][32]byte, withdrawalAddressbytes32 []common.Address, randaoCommitment [][32]byte) (event.Subscription, error) {

	var hashedPubkeyRule []interface{}
	for _, hashedPubkeyItem := range hashedPubkey {
		hashedPubkeyRule = append(hashedPubkeyRule, hashedPubkeyItem)
	}

	var withdrawalAddressbytes32Rule []interface{}
	for _, withdrawalAddressbytes32Item := range withdrawalAddressbytes32 {
		withdrawalAddressbytes32Rule = append(withdrawalAddressbytes32Rule, withdrawalAddressbytes32Item)
	}
	var randaoCommitmentRule []interface{}
	for _, randaoCommitmentItem := range randaoCommitment {
		randaoCommitmentRule = append(randaoCommitmentRule, randaoCommitmentItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.WatchLogs(opts, "ValidatorRegistered", hashedPubkeyRule, withdrawalAddressbytes32Rule, randaoCommitmentRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorRegistrationValidatorRegistered)
				if err := _ValidatorRegistration.contract.UnpackLog(event, "ValidatorRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
