// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// ValidatorRegistrationABI is the input ABI used to generate the binding from.
const ValidatorRegistrationABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"usedPubkey\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"VALIDATOR_DEPOSIT\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_pubkey\",\"type\":\"bytes32\"},{\"name\":\"_withdrawalShardID\",\"type\":\"uint256\"},{\"name\":\"_withdrawalAddressbytes32\",\"type\":\"address\"},{\"name\":\"_randaoCommitment\",\"type\":\"bytes32\"}],\"name\":\"deposit\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"pubKey\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"withdrawalShardID\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"withdrawalAddressbytes32\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"randaoCommitment\",\"type\":\"bytes32\"}],\"name\":\"ValidatorRegistered\",\"type\":\"event\"}]"

// ValidatorRegistrationBin is the compiled bytecode used for deploying new contracts.
const ValidatorRegistrationBin = `0x608060405234801561001057600080fd5b506101c7806100206000396000f3006080604052600436106100565763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166301110845811461005b578063441d92cc14610087578063881d2135146100ae575b600080fd5b34801561006757600080fd5b506100736004356100da565b604080519115158252519081900360200190f35b34801561009357600080fd5b5061009c6100ef565b60408051918252519081900360200190f35b6100d860043560243573ffffffffffffffffffffffffffffffffffffffff604435166064356100fc565b005b60006020819052908152604090205460ff1681565b6801bc16d674ec80000081565b346801bc16d674ec8000001461011157600080fd5b60008481526020819052604090205460ff161561012d57600080fd5b60008481526020818152604091829020805460ff1916600117905581518581529151839273ffffffffffffffffffffffffffffffffffffffff86169288927f7b0678aab009b61a805f5004869728b53a444f9a3e6bb9e22b8537c89af512749281900390910190a4505050505600a165627a7a7230582010cee12b801046464a3d4ffaad0081c2c3cf734f886a556d1e3b4f3565b649380029`

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

// UsedPubkey is a free data retrieval call binding the contract method 0x01110845.
//
// Solidity: function usedPubkey( bytes32) constant returns(bool)
func (_ValidatorRegistration *ValidatorRegistrationCaller) UsedPubkey(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "usedPubkey", arg0)
	return *ret0, err
}

// UsedPubkey is a free data retrieval call binding the contract method 0x01110845.
//
// Solidity: function usedPubkey( bytes32) constant returns(bool)
func (_ValidatorRegistration *ValidatorRegistrationSession) UsedPubkey(arg0 [32]byte) (bool, error) {
	return _ValidatorRegistration.Contract.UsedPubkey(&_ValidatorRegistration.CallOpts, arg0)
}

// UsedPubkey is a free data retrieval call binding the contract method 0x01110845.
//
// Solidity: function usedPubkey( bytes32) constant returns(bool)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) UsedPubkey(arg0 [32]byte) (bool, error) {
	return _ValidatorRegistration.Contract.UsedPubkey(&_ValidatorRegistration.CallOpts, arg0)
}

// Deposit is a paid mutator transaction binding the contract method 0x881d2135.
//
// Solidity: function deposit(_pubkey bytes32, _withdrawalShardID uint256, _withdrawalAddressbytes32 address, _randaoCommitment bytes32) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactor) Deposit(opts *bind.TransactOpts, _pubkey [32]byte, _withdrawalShardID *big.Int, _withdrawalAddressbytes32 common.Address, _randaoCommitment [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.contract.Transact(opts, "deposit", _pubkey, _withdrawalShardID, _withdrawalAddressbytes32, _randaoCommitment)
}

// Deposit is a paid mutator transaction binding the contract method 0x881d2135.
//
// Solidity: function deposit(_pubkey bytes32, _withdrawalShardID uint256, _withdrawalAddressbytes32 address, _randaoCommitment bytes32) returns()
func (_ValidatorRegistration *ValidatorRegistrationSession) Deposit(_pubkey [32]byte, _withdrawalShardID *big.Int, _withdrawalAddressbytes32 common.Address, _randaoCommitment [32]byte) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.Deposit(&_ValidatorRegistration.TransactOpts, _pubkey, _withdrawalShardID, _withdrawalAddressbytes32, _randaoCommitment)
}

// Deposit is a paid mutator transaction binding the contract method 0x881d2135.
//
// Solidity: function deposit(_pubkey bytes32, _withdrawalShardID uint256, _withdrawalAddressbytes32 address, _randaoCommitment bytes32) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactorSession) Deposit(_pubkey [32]byte, _withdrawalShardID *big.Int, _withdrawalAddressbytes32 common.Address, _randaoCommitment [32]byte) (*types.Transaction, error) {
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
	PubKey                   [32]byte
	WithdrawalShardID        *big.Int
	WithdrawalAddressbytes32 common.Address
	RandaoCommitment         [32]byte
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterValidatorRegistered is a free log retrieval operation binding the contract event 0x7b0678aab009b61a805f5004869728b53a444f9a3e6bb9e22b8537c89af51274.
//
// Solidity: event ValidatorRegistered(pubKey indexed bytes32, withdrawalShardID uint256, withdrawalAddressbytes32 indexed address, randaoCommitment indexed bytes32)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) FilterValidatorRegistered(opts *bind.FilterOpts, pubKey [][32]byte, withdrawalAddressbytes32 []common.Address, randaoCommitment [][32]byte) (*ValidatorRegistrationValidatorRegisteredIterator, error) {

	var pubKeyRule []interface{}
	for _, pubKeyItem := range pubKey {
		pubKeyRule = append(pubKeyRule, pubKeyItem)
	}

	var withdrawalAddressbytes32Rule []interface{}
	for _, withdrawalAddressbytes32Item := range withdrawalAddressbytes32 {
		withdrawalAddressbytes32Rule = append(withdrawalAddressbytes32Rule, withdrawalAddressbytes32Item)
	}
	var randaoCommitmentRule []interface{}
	for _, randaoCommitmentItem := range randaoCommitment {
		randaoCommitmentRule = append(randaoCommitmentRule, randaoCommitmentItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.FilterLogs(opts, "ValidatorRegistered", pubKeyRule, withdrawalAddressbytes32Rule, randaoCommitmentRule)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationValidatorRegisteredIterator{contract: _ValidatorRegistration.contract, event: "ValidatorRegistered", logs: logs, sub: sub}, nil
}

// WatchValidatorRegistered is a free log subscription operation binding the contract event 0x7b0678aab009b61a805f5004869728b53a444f9a3e6bb9e22b8537c89af51274.
//
// Solidity: event ValidatorRegistered(pubKey indexed bytes32, withdrawalShardID uint256, withdrawalAddressbytes32 indexed address, randaoCommitment indexed bytes32)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) WatchValidatorRegistered(opts *bind.WatchOpts, sink chan<- *ValidatorRegistrationValidatorRegistered, pubKey [][32]byte, withdrawalAddressbytes32 []common.Address, randaoCommitment [][32]byte) (event.Subscription, error) {

	var pubKeyRule []interface{}
	for _, pubKeyItem := range pubKey {
		pubKeyRule = append(pubKeyRule, pubKeyItem)
	}

	var withdrawalAddressbytes32Rule []interface{}
	for _, withdrawalAddressbytes32Item := range withdrawalAddressbytes32 {
		withdrawalAddressbytes32Rule = append(withdrawalAddressbytes32Rule, withdrawalAddressbytes32Item)
	}
	var randaoCommitmentRule []interface{}
	for _, randaoCommitmentItem := range randaoCommitment {
		randaoCommitmentRule = append(randaoCommitmentRule, randaoCommitmentItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.WatchLogs(opts, "ValidatorRegistered", pubKeyRule, withdrawalAddressbytes32Rule, randaoCommitmentRule)
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
