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
const ValidatorRegistrationABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"MIN_TOPUP_SIZE\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GWEI_PER_ETH\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"MERKLE_TREE_DEPTH\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"DEPOSIT_SIZE\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getReceiptRoot\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"receiptTree\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"SECONDS_PER_DAY\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"depositParams\",\"type\":\"bytes\"}],\"name\":\"deposit\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"fullDepositCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"DEPOSITS_FOR_CHAIN_START\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalDepositCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousReceiptRoot\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"data\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"totalDepositcount\",\"type\":\"uint256\"}],\"name\":\"HashChainValue\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"receiptRoot\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"time\",\"type\":\"bytes\"}],\"name\":\"ChainStart\",\"type\":\"event\"}]"

// ValidatorRegistrationBin is the compiled bytecode used for deploying new contracts.
const ValidatorRegistrationBin = `608060405234801561001057600080fd5b50610c20806100206000396000f3fe6080604052600436106100a8577c0100000000000000000000000000000000000000000000000000000000600035046320d24cb781146100ad57806324eda209146100d45780633568cda0146100e957806336bf3325146100fe5780634213155f14610113578063701f82121461019d57806374f0314f146101c757806398b1e06a146101dc578063c78598c414610284578063cac0057c14610299578063d6343867146102ae575b600080fd5b3480156100b957600080fd5b506100c26102c3565b60408051918252519081900360200190f35b3480156100e057600080fd5b506100c26102cf565b3480156100f557600080fd5b506100c26102d7565b34801561010a57600080fd5b506100c26102dc565b34801561011f57600080fd5b506101286102e9565b6040805160208082528351818301528351919283929083019185019080838360005b8381101561016257818101518382015260200161014a565b50505050905090810190601f16801561018f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156101a957600080fd5b50610128600480360360208110156101c057600080fd5b50356103a5565b3480156101d357600080fd5b506100c261043f565b610282600480360360208110156101f257600080fd5b81019060208101813564010000000081111561020d57600080fd5b82018360208201111561021f57600080fd5b8035906020019184600183028401116401000000008311171561024157600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250929550610446945050505050565b005b34801561029057600080fd5b506100c2610b4b565b3480156102a557600080fd5b506100c2610b51565b3480156102ba57600080fd5b506100c2610b56565b670de0b6b3a764000081565b633b9aca0081565b601081565b6801bc16d674ec80000081565b6001600081815260209081527fada5013122d395ba3c54772283fb069b10426056ef8ca54750cb9bb552a59e7d805460408051600295831615610100026000190190921694909404601f8101849004840282018401909452838152606093909283018282801561039a5780601f1061036f5761010080835404028352916020019161039a565b820191906000526020600020905b81548152906001019060200180831161037d57829003601f168201915b505050505090505b90565b600060208181529181526040908190208054825160026001831615610100026000190190921691909104601f8101859004850282018501909352828152929091908301828280156104375780601f1061040c57610100808354040283529160200191610437565b820191906000526020600020905b81548152906001019060200180831161041a57829003601f168201915b505050505081565b6201518081565b6801bc16d674ec8000003411156104e457604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f4465706f7369742063616e27742062652067726561746572207468616e20444560448201527f504f5349545f53495a452e000000000000000000000000000000000000000000606482015290519081900360840190fd5b670de0b6b3a764000034101561058157604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f4465706f7369742063616e2774206265206c6573736572207468616e204d494e60448201527f5f544f5055505f53495a452e0000000000000000000000000000000000000000606482015290519081900360840190fd5b60025460408051780100000000000000000000000000000000000000000000000067ffffffffffffffff633b9aca003404811682026020808501919091528451808503600801815260288501865242909216909202604884015283516030818503018152605084019094528051620100009095019490939260609285928592899260709091019182918601908083835b602083106106305780518252601f199092019160209182019101610611565b51815160209384036101000a600019018019909216911617905286519190930192860191508083835b602083106106785780518252601f199092019160209182019101610659565b51815160209384036101000a600019018019909216911617905285519190930192850191508083835b602083106106c05780518252601f1990920191602091820191016106a1565b6001836020036101000a038019825116818451168082178552505050505050905001935050505060405160208183030381529060405290506000806001815260200190815260200160002060405180828054600181600116156101000203166002900480156107665780601f10610744576101008083540402835291820191610766565b820191906000526020600020905b815481529060010190602001808311610752575b505091505060405180910390207fd60002d8f2bba463e7d533a8fa9b84d9d7451dc0bb7e9da868104601a5d2b9ef826002546040518080602001838152602001828103825284818151815260200191508051906020019080838360005b838110156107db5781810151838201526020016107c3565b50505050905090810190601f1680156108085780820380516001836020036101000a031916815260200191505b50935050505060405180910390a2805160208083019190912060408051808401929092528051808303840181529181018152600087815280845220815161085493919290910190610b5c565b5060005b601081101561099f5760028504945060008086600202815260200190815260200160002060008087600202600101815260200190815260200160002060405160200180838054600181600116156101000203166002900480156108f25780601f106108d05761010080835404028352918201916108f2565b820191906000526020600020905b8154815290600101906020018083116108de575b50508280546001816001161561010002031660029004801561094b5780601f1061092957610100808354040283529182019161094b565b820191906000526020600020905b815481529060010190602001808311610937575b505060408051601f19818403018152828252805160209182012081840152815180840382018152928201825260008b8152808252919091208251610996965090945091019150610b5c565b50600101610858565b50600280546001019055346801bc16d674ec8000001415610b44576001805481019081905560081415610b44576000620151808042064203019050606081604051602001808267ffffffffffffffff1667ffffffffffffffff1678010000000000000000000000000000000000000000000000000281526008019150506040516020818303038152906040529050600080600181526020019081526020016000206040518082805460018160011615610100020316600290048015610a9b5780601f10610a79576101008083540402835291820191610a9b565b820191906000526020600020905b815481529060010190602001808311610a87575b505091505060405180910390207fb24eb1954aa12b3433be2b11740f2f73c8673453064575f93f223a85f1215f3a826040518080602001828103825283818151815260200191508051906020019080838360005b83811015610b07578181015183820152602001610aef565b50505050905090810190601f168015610b345780820380516001836020036101000a031916815260200191505b509250505060405180910390a250505b5050505050565b60015481565b600881565b60025481565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10610b9d57805160ff1916838001178555610bca565b82800160010185558215610bca579182015b82811115610bca578251825591602001919060010190610baf565b50610bd6929150610bda565b5090565b6103a291905b80821115610bd65760008155600101610be056fea165627a7a723058206508486d04f5b069ced869a8b6a1758ca44c9bf17b58ae34a733c2870620b1f00029`

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

// DEPOSITSFORCHAINSTART is a free data retrieval call binding the contract method 0xcac0057c.
//
// Solidity: function DEPOSITS_FOR_CHAIN_START() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) DEPOSITSFORCHAINSTART(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "DEPOSITS_FOR_CHAIN_START")
	return *ret0, err
}

// DEPOSITSFORCHAINSTART is a free data retrieval call binding the contract method 0xcac0057c.
//
// Solidity: function DEPOSITS_FOR_CHAIN_START() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) DEPOSITSFORCHAINSTART() (*big.Int, error) {
	return _ValidatorRegistration.Contract.DEPOSITSFORCHAINSTART(&_ValidatorRegistration.CallOpts)
}

// DEPOSITSFORCHAINSTART is a free data retrieval call binding the contract method 0xcac0057c.
//
// Solidity: function DEPOSITS_FOR_CHAIN_START() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) DEPOSITSFORCHAINSTART() (*big.Int, error) {
	return _ValidatorRegistration.Contract.DEPOSITSFORCHAINSTART(&_ValidatorRegistration.CallOpts)
}

// DEPOSITSIZE is a free data retrieval call binding the contract method 0x36bf3325.
//
// Solidity: function DEPOSIT_SIZE() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) DEPOSITSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "DEPOSIT_SIZE")
	return *ret0, err
}

// DEPOSITSIZE is a free data retrieval call binding the contract method 0x36bf3325.
//
// Solidity: function DEPOSIT_SIZE() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) DEPOSITSIZE() (*big.Int, error) {
	return _ValidatorRegistration.Contract.DEPOSITSIZE(&_ValidatorRegistration.CallOpts)
}

// DEPOSITSIZE is a free data retrieval call binding the contract method 0x36bf3325.
//
// Solidity: function DEPOSIT_SIZE() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) DEPOSITSIZE() (*big.Int, error) {
	return _ValidatorRegistration.Contract.DEPOSITSIZE(&_ValidatorRegistration.CallOpts)
}

// GWEIPERETH is a free data retrieval call binding the contract method 0x24eda209.
//
// Solidity: function GWEI_PER_ETH() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) GWEIPERETH(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "GWEI_PER_ETH")
	return *ret0, err
}

// GWEIPERETH is a free data retrieval call binding the contract method 0x24eda209.
//
// Solidity: function GWEI_PER_ETH() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) GWEIPERETH() (*big.Int, error) {
	return _ValidatorRegistration.Contract.GWEIPERETH(&_ValidatorRegistration.CallOpts)
}

// GWEIPERETH is a free data retrieval call binding the contract method 0x24eda209.
//
// Solidity: function GWEI_PER_ETH() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) GWEIPERETH() (*big.Int, error) {
	return _ValidatorRegistration.Contract.GWEIPERETH(&_ValidatorRegistration.CallOpts)
}

// MERKLETREEDEPTH is a free data retrieval call binding the contract method 0x3568cda0.
//
// Solidity: function MERKLE_TREE_DEPTH() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) MERKLETREEDEPTH(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "MERKLE_TREE_DEPTH")
	return *ret0, err
}

// MERKLETREEDEPTH is a free data retrieval call binding the contract method 0x3568cda0.
//
// Solidity: function MERKLE_TREE_DEPTH() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) MERKLETREEDEPTH() (*big.Int, error) {
	return _ValidatorRegistration.Contract.MERKLETREEDEPTH(&_ValidatorRegistration.CallOpts)
}

// MERKLETREEDEPTH is a free data retrieval call binding the contract method 0x3568cda0.
//
// Solidity: function MERKLE_TREE_DEPTH() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) MERKLETREEDEPTH() (*big.Int, error) {
	return _ValidatorRegistration.Contract.MERKLETREEDEPTH(&_ValidatorRegistration.CallOpts)
}

// MINTOPUPSIZE is a free data retrieval call binding the contract method 0x20d24cb7.
//
// Solidity: function MIN_TOPUP_SIZE() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) MINTOPUPSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "MIN_TOPUP_SIZE")
	return *ret0, err
}

// MINTOPUPSIZE is a free data retrieval call binding the contract method 0x20d24cb7.
//
// Solidity: function MIN_TOPUP_SIZE() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) MINTOPUPSIZE() (*big.Int, error) {
	return _ValidatorRegistration.Contract.MINTOPUPSIZE(&_ValidatorRegistration.CallOpts)
}

// MINTOPUPSIZE is a free data retrieval call binding the contract method 0x20d24cb7.
//
// Solidity: function MIN_TOPUP_SIZE() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) MINTOPUPSIZE() (*big.Int, error) {
	return _ValidatorRegistration.Contract.MINTOPUPSIZE(&_ValidatorRegistration.CallOpts)
}

// SECONDSPERDAY is a free data retrieval call binding the contract method 0x74f0314f.
//
// Solidity: function SECONDS_PER_DAY() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) SECONDSPERDAY(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "SECONDS_PER_DAY")
	return *ret0, err
}

// SECONDSPERDAY is a free data retrieval call binding the contract method 0x74f0314f.
//
// Solidity: function SECONDS_PER_DAY() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) SECONDSPERDAY() (*big.Int, error) {
	return _ValidatorRegistration.Contract.SECONDSPERDAY(&_ValidatorRegistration.CallOpts)
}

// SECONDSPERDAY is a free data retrieval call binding the contract method 0x74f0314f.
//
// Solidity: function SECONDS_PER_DAY() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) SECONDSPERDAY() (*big.Int, error) {
	return _ValidatorRegistration.Contract.SECONDSPERDAY(&_ValidatorRegistration.CallOpts)
}

// FullDepositCount is a free data retrieval call binding the contract method 0xc78598c4.
//
// Solidity: function fullDepositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) FullDepositCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "fullDepositCount")
	return *ret0, err
}

// FullDepositCount is a free data retrieval call binding the contract method 0xc78598c4.
//
// Solidity: function fullDepositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) FullDepositCount() (*big.Int, error) {
	return _ValidatorRegistration.Contract.FullDepositCount(&_ValidatorRegistration.CallOpts)
}

// FullDepositCount is a free data retrieval call binding the contract method 0xc78598c4.
//
// Solidity: function fullDepositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) FullDepositCount() (*big.Int, error) {
	return _ValidatorRegistration.Contract.FullDepositCount(&_ValidatorRegistration.CallOpts)
}

// GetReceiptRoot is a free data retrieval call binding the contract method 0x4213155f.
//
// Solidity: function getReceiptRoot() constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCaller) GetReceiptRoot(opts *bind.CallOpts) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "getReceiptRoot")
	return *ret0, err
}

// GetReceiptRoot is a free data retrieval call binding the contract method 0x4213155f.
//
// Solidity: function getReceiptRoot() constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationSession) GetReceiptRoot() ([]byte, error) {
	return _ValidatorRegistration.Contract.GetReceiptRoot(&_ValidatorRegistration.CallOpts)
}

// GetReceiptRoot is a free data retrieval call binding the contract method 0x4213155f.
//
// Solidity: function getReceiptRoot() constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) GetReceiptRoot() ([]byte, error) {
	return _ValidatorRegistration.Contract.GetReceiptRoot(&_ValidatorRegistration.CallOpts)
}

// ReceiptTree is a free data retrieval call binding the contract method 0x701f8212.
//
// Solidity: function receiptTree( uint256) constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCaller) ReceiptTree(opts *bind.CallOpts, arg0 *big.Int) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "receiptTree", arg0)
	return *ret0, err
}

// ReceiptTree is a free data retrieval call binding the contract method 0x701f8212.
//
// Solidity: function receiptTree( uint256) constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationSession) ReceiptTree(arg0 *big.Int) ([]byte, error) {
	return _ValidatorRegistration.Contract.ReceiptTree(&_ValidatorRegistration.CallOpts, arg0)
}

// ReceiptTree is a free data retrieval call binding the contract method 0x701f8212.
//
// Solidity: function receiptTree( uint256) constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) ReceiptTree(arg0 *big.Int) ([]byte, error) {
	return _ValidatorRegistration.Contract.ReceiptTree(&_ValidatorRegistration.CallOpts, arg0)
}

// TotalDepositCount is a free data retrieval call binding the contract method 0xd6343867.
//
// Solidity: function totalDepositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) TotalDepositCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "totalDepositCount")
	return *ret0, err
}

// TotalDepositCount is a free data retrieval call binding the contract method 0xd6343867.
//
// Solidity: function totalDepositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) TotalDepositCount() (*big.Int, error) {
	return _ValidatorRegistration.Contract.TotalDepositCount(&_ValidatorRegistration.CallOpts)
}

// TotalDepositCount is a free data retrieval call binding the contract method 0xd6343867.
//
// Solidity: function totalDepositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) TotalDepositCount() (*big.Int, error) {
	return _ValidatorRegistration.Contract.TotalDepositCount(&_ValidatorRegistration.CallOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(depositParams bytes) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactor) Deposit(opts *bind.TransactOpts, depositParams []byte) (*types.Transaction, error) {
	return _ValidatorRegistration.contract.Transact(opts, "deposit", depositParams)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(depositParams bytes) returns()
func (_ValidatorRegistration *ValidatorRegistrationSession) Deposit(depositParams []byte) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.Deposit(&_ValidatorRegistration.TransactOpts, depositParams)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(depositParams bytes) returns()
func (_ValidatorRegistration *ValidatorRegistrationTransactorSession) Deposit(depositParams []byte) (*types.Transaction, error) {
	return _ValidatorRegistration.Contract.Deposit(&_ValidatorRegistration.TransactOpts, depositParams)
}

// ValidatorRegistrationChainStartIterator is returned from FilterChainStart and is used to iterate over the raw logs and unpacked data for ChainStart events raised by the ValidatorRegistration contract.
type ValidatorRegistrationChainStartIterator struct {
	Event *ValidatorRegistrationChainStart // Event containing the contract specifics and raw log

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
func (it *ValidatorRegistrationChainStartIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorRegistrationChainStart)
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
		it.Event = new(ValidatorRegistrationChainStart)
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
func (it *ValidatorRegistrationChainStartIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorRegistrationChainStartIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorRegistrationChainStart represents a ChainStart event raised by the ValidatorRegistration contract.
type ValidatorRegistrationChainStart struct {
	ReceiptRoot common.Hash
	Time        []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChainStart is a free log retrieval operation binding the contract event 0xb24eb1954aa12b3433be2b11740f2f73c8673453064575f93f223a85f1215f3a.
//
// Solidity: e ChainStart(receiptRoot indexed bytes, time bytes)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) FilterChainStart(opts *bind.FilterOpts, receiptRoot [][]byte) (*ValidatorRegistrationChainStartIterator, error) {

	var receiptRootRule []interface{}
	for _, receiptRootItem := range receiptRoot {
		receiptRootRule = append(receiptRootRule, receiptRootItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.FilterLogs(opts, "ChainStart", receiptRootRule)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationChainStartIterator{contract: _ValidatorRegistration.contract, event: "ChainStart", logs: logs, sub: sub}, nil
}

// WatchChainStart is a free log subscription operation binding the contract event 0xb24eb1954aa12b3433be2b11740f2f73c8673453064575f93f223a85f1215f3a.
//
// Solidity: e ChainStart(receiptRoot indexed bytes, time bytes)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) WatchChainStart(opts *bind.WatchOpts, sink chan<- *ValidatorRegistrationChainStart, receiptRoot [][]byte) (event.Subscription, error) {

	var receiptRootRule []interface{}
	for _, receiptRootItem := range receiptRoot {
		receiptRootRule = append(receiptRootRule, receiptRootItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.WatchLogs(opts, "ChainStart", receiptRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorRegistrationChainStart)
				if err := _ValidatorRegistration.contract.UnpackLog(event, "ChainStart", log); err != nil {
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

// ValidatorRegistrationHashChainValueIterator is returned from FilterHashChainValue and is used to iterate over the raw logs and unpacked data for HashChainValue events raised by the ValidatorRegistration contract.
type ValidatorRegistrationHashChainValueIterator struct {
	Event *ValidatorRegistrationHashChainValue // Event containing the contract specifics and raw log

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
func (it *ValidatorRegistrationHashChainValueIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorRegistrationHashChainValue)
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
		it.Event = new(ValidatorRegistrationHashChainValue)
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
func (it *ValidatorRegistrationHashChainValueIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorRegistrationHashChainValueIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorRegistrationHashChainValue represents a HashChainValue event raised by the ValidatorRegistration contract.
type ValidatorRegistrationHashChainValue struct {
	PreviousReceiptRoot common.Hash
	Data                []byte
	TotalDepositcount   *big.Int
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterHashChainValue is a free log retrieval operation binding the contract event 0xd60002d8f2bba463e7d533a8fa9b84d9d7451dc0bb7e9da868104601a5d2b9ef.
//
// Solidity: e HashChainValue(previousReceiptRoot indexed bytes, data bytes, totalDepositcount uint256)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) FilterHashChainValue(opts *bind.FilterOpts, previousReceiptRoot [][]byte) (*ValidatorRegistrationHashChainValueIterator, error) {

	var previousReceiptRootRule []interface{}
	for _, previousReceiptRootItem := range previousReceiptRoot {
		previousReceiptRootRule = append(previousReceiptRootRule, previousReceiptRootItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.FilterLogs(opts, "HashChainValue", previousReceiptRootRule)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationHashChainValueIterator{contract: _ValidatorRegistration.contract, event: "HashChainValue", logs: logs, sub: sub}, nil
}

// WatchHashChainValue is a free log subscription operation binding the contract event 0xd60002d8f2bba463e7d533a8fa9b84d9d7451dc0bb7e9da868104601a5d2b9ef.
//
// Solidity: e HashChainValue(previousReceiptRoot indexed bytes, data bytes, totalDepositcount uint256)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) WatchHashChainValue(opts *bind.WatchOpts, sink chan<- *ValidatorRegistrationHashChainValue, previousReceiptRoot [][]byte) (event.Subscription, error) {

	var previousReceiptRootRule []interface{}
	for _, previousReceiptRootItem := range previousReceiptRoot {
		previousReceiptRootRule = append(previousReceiptRootRule, previousReceiptRootItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.WatchLogs(opts, "HashChainValue", previousReceiptRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorRegistrationHashChainValue)
				if err := _ValidatorRegistration.contract.UnpackLog(event, "HashChainValue", log); err != nil {
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
