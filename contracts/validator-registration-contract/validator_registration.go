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
	"github.com/ethereum/go-ethereum/event"
)

// SafeMathABI is the input ABI used to generate the binding from.
const SafeMathABI = "[]"

// SafeMathBin is the compiled bytecode used for deploying new contracts.
const SafeMathBin = `0x604c602c600b82828239805160001a60731460008114601c57601e565bfe5b5030600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea165627a7a72305820954f121d8f4cd45f20ff7158b3f5600d35573b1f53cf9553f114380da71b23980029`

// DeploySafeMath deploys a new Ethereum contract, binding an instance of SafeMath to it.
func DeploySafeMath(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SafeMath, error) {
	parsed, err := abi.JSON(strings.NewReader(SafeMathABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SafeMathBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SafeMath{SafeMathCaller: SafeMathCaller{contract: contract}, SafeMathTransactor: SafeMathTransactor{contract: contract}, SafeMathFilterer: SafeMathFilterer{contract: contract}}, nil
}

// SafeMath is an auto generated Go binding around an Ethereum contract.
type SafeMath struct {
	SafeMathCaller     // Read-only binding to the contract
	SafeMathTransactor // Write-only binding to the contract
	SafeMathFilterer   // Log filterer for contract events
}

// SafeMathCaller is an auto generated read-only Go binding around an Ethereum contract.
type SafeMathCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SafeMathTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SafeMathFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SafeMathSession struct {
	Contract     *SafeMath         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SafeMathCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SafeMathCallerSession struct {
	Contract *SafeMathCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// SafeMathTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SafeMathTransactorSession struct {
	Contract     *SafeMathTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// SafeMathRaw is an auto generated low-level Go binding around an Ethereum contract.
type SafeMathRaw struct {
	Contract *SafeMath // Generic contract binding to access the raw methods on
}

// SafeMathCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SafeMathCallerRaw struct {
	Contract *SafeMathCaller // Generic read-only contract binding to access the raw methods on
}

// SafeMathTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SafeMathTransactorRaw struct {
	Contract *SafeMathTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSafeMath creates a new instance of SafeMath, bound to a specific deployed contract.
func NewSafeMath(address common.Address, backend bind.ContractBackend) (*SafeMath, error) {
	contract, err := bindSafeMath(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SafeMath{SafeMathCaller: SafeMathCaller{contract: contract}, SafeMathTransactor: SafeMathTransactor{contract: contract}, SafeMathFilterer: SafeMathFilterer{contract: contract}}, nil
}

// NewSafeMathCaller creates a new read-only instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathCaller(address common.Address, caller bind.ContractCaller) (*SafeMathCaller, error) {
	contract, err := bindSafeMath(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SafeMathCaller{contract: contract}, nil
}

// NewSafeMathTransactor creates a new write-only instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathTransactor(address common.Address, transactor bind.ContractTransactor) (*SafeMathTransactor, error) {
	contract, err := bindSafeMath(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SafeMathTransactor{contract: contract}, nil
}

// NewSafeMathFilterer creates a new log filterer instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathFilterer(address common.Address, filterer bind.ContractFilterer) (*SafeMathFilterer, error) {
	contract, err := bindSafeMath(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SafeMathFilterer{contract: contract}, nil
}

// bindSafeMath binds a generic wrapper to an already deployed contract.
func bindSafeMath(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SafeMathABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeMath *SafeMathRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SafeMath.Contract.SafeMathCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeMath *SafeMathRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeMath.Contract.SafeMathTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeMath *SafeMathRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeMath.Contract.SafeMathTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeMath *SafeMathCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SafeMath.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeMath *SafeMathTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeMath.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeMath *SafeMathTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeMath.Contract.contract.Transact(opts, method, params...)
}

// ValidatorRegistrationABI is the input ABI used to generate the binding from.
const ValidatorRegistrationABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"MIN_TOPUP_SIZE\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GWEI_PER_ETH\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"depositCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"MERKLE_TREE_DEPTH\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"DEPOSIT_SIZE\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getDepositRoot\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"SECONDS_PER_DAY\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"depositParams\",\"type\":\"bytes\"}],\"name\":\"deposit\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"fullDepositCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"DEPOSITS_FOR_CHAIN_START\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"depositTree\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousReceiptRoot\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"data\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"merkleTreeIndex\",\"type\":\"bytes\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"receiptRoot\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"time\",\"type\":\"bytes\"}],\"name\":\"ChainStart\",\"type\":\"event\"}]"

// ValidatorRegistrationBin is the compiled bytecode used for deploying new contracts.
const ValidatorRegistrationBin = `0x608060405234801561001057600080fd5b50610e4b806100206000396000f3fe6080604052600436106100b9576000357c0100000000000000000000000000000000000000000000000000000000900480633ae05047116100815780633ae050471461013957806374f0314f146101c357806398b1e06a146101d8578063c78598c414610280578063cac0057c14610295578063f8a5841b146102aa576100b9565b806320d24cb7146100be57806324eda209146100e55780632dfdf0b5146100fa5780633568cda01461010f57806336bf332514610124575b600080fd5b3480156100ca57600080fd5b506100d36102d4565b60408051918252519081900360200190f35b3480156100f157600080fd5b506100d36102e0565b34801561010657600080fd5b506100d36102e8565b34801561011b57600080fd5b506100d36102ee565b34801561013057600080fd5b506100d36102f3565b34801561014557600080fd5b5061014e610300565b6040805160208082528351818301528351919283929083019185019080838360005b83811015610188578181015183820152602001610170565b50505050905090810190601f1680156101b55780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156101cf57600080fd5b506100d36103bc565b61027e600480360360208110156101ee57600080fd5b81019060208101813564010000000081111561020957600080fd5b82018360208201111561021b57600080fd5b8035906020019184600183028401116401000000008311171561023d57600080fd5b91908080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152509295506103c3945050505050565b005b34801561028c57600080fd5b506100d3610afc565b3480156102a157600080fd5b506100d3610b02565b3480156102b657600080fd5b5061014e600480360360208110156102cd57600080fd5b5035610b07565b670de0b6b3a764000081565b633b9aca0081565b60015481565b601081565b6801bc16d674ec80000081565b6001600081815260209081527fada5013122d395ba3c54772283fb069b10426056ef8ca54750cb9bb552a59e7d805460408051600295831615610100026000190190921694909404601f810184900484028201840190945283815260609390928301828280156103b15780601f10610386576101008083540402835291602001916103b1565b820191906000526020600020905b81548152906001019060200180831161039457829003601f168201915b505050505090505b90565b6201518081565b6801bc16d674ec80000034111561040e5760405160e560020a62461bcd02815260040180806020018281038252602b815260200180610df5602b913960400191505060405180910390fd5b670de0b6b3a76400003410156104585760405160e560020a62461bcd02815260040180806020018281038252602c815260200180610dc9602c913960400191505060405180910390fd5b60015460408051780100000000000000000000000000000000000000000000000067ffffffffffffffff633b9aca003404811682026020808501919091528451808503600801815260288501865242909216909202604884015283516030818503018152605084019094528051620100009095019490939260609285928592899260709091019182918601908083835b602083106105075780518252601f1990920191602091820191016104e8565b51815160209384036101000a600019018019909216911617905286519190930192860191508083835b6020831061054f5780518252601f199092019160209182019101610530565b51815160209384036101000a600019018019909216911617905285519190930192850191508083835b602083106105975780518252601f199092019160209182019101610578565b6001836020036101000a03801982511681845116808217855250505050505090500193505050506040516020818303038152906040529050606084604051602001808267ffffffffffffffff1667ffffffffffffffff16780100000000000000000000000000000000000000000000000002815260080191505060405160208183030381529060405290506000806001815260200190815260200160002060405180828054600181600116156101000203166002900480156106905780601f1061066e576101008083540402835291820191610690565b820191906000526020600020905b81548152906001019060200180831161067c575b505091505060405180910390207f74ec17dbc2e96aa26ff3c11cef381b6dfbbeda92eb30c6f09b096abd9aef469b8383604051808060200180602001838103835285818151815260200191508051906020019080838360005b838110156107015781810151838201526020016106e9565b50505050905090810190601f16801561072e5780820380516001836020036101000a031916815260200191505b50838103825284518152845160209182019186019080838360005b83811015610761578181015183820152602001610749565b50505050905090810190601f16801561078e5780820380516001836020036101000a031916815260200191505b5094505050505060405180910390a281516020808401919091206040805180840192909252805180830384018152918101815260008881528084522081516107db93919290910190610c8f565b5060005b60108110156109265760028604955060008087600202815260200190815260200160002060008088600202600101815260200190815260200160002060405160200180838054600181600116156101000203166002900480156108795780601f10610857576101008083540402835291820191610879565b820191906000526020600020905b815481529060010190602001808311610865575b5050828054600181600116156101000203166002900480156108d25780601f106108b05761010080835404028352918201916108d2565b820191906000526020600020905b8154815290600101906020018083116108be575b505060408051601f19818403018152828252805160209182012081840152815180840382018152928201825260008c815280825291909120825161091d965090945091019150610c8f565b506001016107df565b506001805481019055346801bc16d674ec8000001415610af457600280546001019081905560081415610af45760006109886201518061097c81610970428063ffffffff610ba116565b9063ffffffff610beb16565b9063ffffffff610c4116565b9050606081604051602001808267ffffffffffffffff1667ffffffffffffffff1678010000000000000000000000000000000000000000000000000281526008019150506040516020818303038152906040529050600080600181526020019081526020016000206040518082805460018160011615610100020316600290048015610a4b5780601f10610a29576101008083540402835291820191610a4b565b820191906000526020600020905b815481529060010190602001808311610a37575b505091505060405180910390207fb24eb1954aa12b3433be2b11740f2f73c8673453064575f93f223a85f1215f3a826040518080602001828103825283818151815260200191508051906020019080838360005b83811015610ab7578181015183820152602001610a9f565b50505050905090810190601f168015610ae45780820380516001836020036101000a031916815260200191505b509250505060405180910390a250505b505050505050565b60025481565b600881565b600060208181529181526040908190208054825160026001831615610100026000190190921691909104601f810185900485028201850190935282815292909190830182828015610b995780601f10610b6e57610100808354040283529160200191610b99565b820191906000526020600020905b815481529060010190602001808311610b7c57829003601f168201915b505050505081565b600082821115610be55760405160e560020a62461bcd028152600401808060200182810382526037815260200180610d926037913960400191505060405180910390fd5b50900390565b6000811515610c2e5760405160e560020a62461bcd028152600401808060200182810382526024815260200180610d286024913960400191505060405180910390fd5b8183811515610c3957fe5b069392505050565b600082820183811015610c885760405160e560020a62461bcd028152600401808060200182810382526046815260200180610d4c6046913960600191505060405180910390fd5b9392505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10610cd057805160ff1916838001178555610cfd565b82800160010185558215610cfd579182015b82811115610cfd578251825591602001919060010190610ce2565b50610d09929150610d0d565b5090565b6103b991905b80821115610d095760008155600101610d1356fe546865207365636f6e6420706172616d657465722063616e206e6f74206265207a65726f54686520666972737420706172616d65746572742063616e206e6f742062652067726561746572207468616e207468652073756d206f662074776f20706172616d6574657273546865207365636f6e6420706172616d657465722063616e206e6f742062652067726561746572207468616e206669727374206f6e652e4465706f7369742063616e2774206265206c6573736572207468616e204d494e5f544f5055505f53495a452e4465706f7369742063616e27742062652067726561746572207468616e204445504f5349545f53495a452ea165627a7a72305820bf3714b406e649067cee4203a488d956bd49bcd2ea8ee803059a92346d0c39580029`

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

// DepositCount is a free data retrieval call binding the contract method 0x2dfdf0b5.
//
// Solidity: function depositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCaller) DepositCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "depositCount")
	return *ret0, err
}

// DepositCount is a free data retrieval call binding the contract method 0x2dfdf0b5.
//
// Solidity: function depositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationSession) DepositCount() (*big.Int, error) {
	return _ValidatorRegistration.Contract.DepositCount(&_ValidatorRegistration.CallOpts)
}

// DepositCount is a free data retrieval call binding the contract method 0x2dfdf0b5.
//
// Solidity: function depositCount() constant returns(uint256)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) DepositCount() (*big.Int, error) {
	return _ValidatorRegistration.Contract.DepositCount(&_ValidatorRegistration.CallOpts)
}

// DepositTree is a free data retrieval call binding the contract method 0xf8a5841b.
//
// Solidity: function depositTree( uint256) constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCaller) DepositTree(opts *bind.CallOpts, arg0 *big.Int) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "depositTree", arg0)
	return *ret0, err
}

// DepositTree is a free data retrieval call binding the contract method 0xf8a5841b.
//
// Solidity: function depositTree( uint256) constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationSession) DepositTree(arg0 *big.Int) ([]byte, error) {
	return _ValidatorRegistration.Contract.DepositTree(&_ValidatorRegistration.CallOpts, arg0)
}

// DepositTree is a free data retrieval call binding the contract method 0xf8a5841b.
//
// Solidity: function depositTree( uint256) constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) DepositTree(arg0 *big.Int) ([]byte, error) {
	return _ValidatorRegistration.Contract.DepositTree(&_ValidatorRegistration.CallOpts, arg0)
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

// GetDepositRoot is a free data retrieval call binding the contract method 0x3ae05047.
//
// Solidity: function getDepositRoot() constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCaller) GetDepositRoot(opts *bind.CallOpts) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _ValidatorRegistration.contract.Call(opts, out, "getDepositRoot")
	return *ret0, err
}

// GetDepositRoot is a free data retrieval call binding the contract method 0x3ae05047.
//
// Solidity: function getDepositRoot() constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationSession) GetDepositRoot() ([]byte, error) {
	return _ValidatorRegistration.Contract.GetDepositRoot(&_ValidatorRegistration.CallOpts)
}

// GetDepositRoot is a free data retrieval call binding the contract method 0x3ae05047.
//
// Solidity: function getDepositRoot() constant returns(bytes)
func (_ValidatorRegistration *ValidatorRegistrationCallerSession) GetDepositRoot() ([]byte, error) {
	return _ValidatorRegistration.Contract.GetDepositRoot(&_ValidatorRegistration.CallOpts)
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

// ValidatorRegistrationDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the ValidatorRegistration contract.
type ValidatorRegistrationDepositIterator struct {
	Event *ValidatorRegistrationDeposit // Event containing the contract specifics and raw log

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
func (it *ValidatorRegistrationDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorRegistrationDeposit)
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
		it.Event = new(ValidatorRegistrationDeposit)
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
func (it *ValidatorRegistrationDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorRegistrationDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorRegistrationDeposit represents a Deposit event raised by the ValidatorRegistration contract.
type ValidatorRegistrationDeposit struct {
	PreviousReceiptRoot common.Hash
	Data                []byte
	MerkleTreeIndex     []byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0x74ec17dbc2e96aa26ff3c11cef381b6dfbbeda92eb30c6f09b096abd9aef469b.
//
// Solidity: e Deposit(previousReceiptRoot indexed bytes, data bytes, merkleTreeIndex bytes)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) FilterDeposit(opts *bind.FilterOpts, previousReceiptRoot [][]byte) (*ValidatorRegistrationDepositIterator, error) {

	var previousReceiptRootRule []interface{}
	for _, previousReceiptRootItem := range previousReceiptRoot {
		previousReceiptRootRule = append(previousReceiptRootRule, previousReceiptRootItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.FilterLogs(opts, "Deposit", previousReceiptRootRule)
	if err != nil {
		return nil, err
	}
	return &ValidatorRegistrationDepositIterator{contract: _ValidatorRegistration.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0x74ec17dbc2e96aa26ff3c11cef381b6dfbbeda92eb30c6f09b096abd9aef469b.
//
// Solidity: e Deposit(previousReceiptRoot indexed bytes, data bytes, merkleTreeIndex bytes)
func (_ValidatorRegistration *ValidatorRegistrationFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *ValidatorRegistrationDeposit, previousReceiptRoot [][]byte) (event.Subscription, error) {

	var previousReceiptRootRule []interface{}
	for _, previousReceiptRootItem := range previousReceiptRoot {
		previousReceiptRootRule = append(previousReceiptRootRule, previousReceiptRootItem)
	}

	logs, sub, err := _ValidatorRegistration.contract.WatchLogs(opts, "Deposit", previousReceiptRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorRegistrationDeposit)
				if err := _ValidatorRegistration.contract.UnpackLog(event, "Deposit", log); err != nil {
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
