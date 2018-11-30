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

// ValidatorRegistrationABI is the input ABI used to generate the binding from.
const ValidatorRegistrationABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"MIN_TOPUP_SIZE\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GWEI_PER_ETH\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"MERKLE_TREE_DEPTH\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"DEPOSIT_SIZE\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getReceiptRoot\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"receiptTree\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"SECONDS_PER_DAY\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"depositParams\",\"type\":\"bytes\"}],\"name\":\"deposit\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"DEPOSITS_FOR_CHAIN_START\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalDepositCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousReceiptRoot\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"data\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"totalDepositcount\",\"type\":\"uint256\"}],\"name\":\"HashChainValue\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"receiptRoot\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"time\",\"type\":\"bytes\"}],\"name\":\"ChainStart\",\"type\":\"event\"}]"

// ValidatorRegistrationBin is the compiled bytecode used for deploying new contracts.
const ValidatorRegistrationBin = `0x608060405234801561001057600080fd5b50610c49806100206000396000f3006080604052600436106100a35763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166320d24cb781146100a857806324eda209146100d35780633568cda0146100e857806336bf3325146100fd5780634213155f14610112578063701f82121461013457806374f0314f1461015457806398b1e06a14610169578063cac0057c1461017e578063d634386714610193575b600080fd5b3480156100b457600080fd5b506100bd6101a8565b6040516100ca9190610b68565b60405180910390f35b3480156100df57600080fd5b506100bd6101b4565b3480156100f457600080fd5b506100bd6101bc565b34801561010957600080fd5b506100bd6101c1565b34801561011e57600080fd5b506101276101ce565b6040516100ca9190610b11565b34801561014057600080fd5b5061012761014f366004610a03565b61028a565b34801561016057600080fd5b506100bd610324565b61017c6101773660046109c6565b61032b565b005b34801561018a57600080fd5b506100bd610813565b34801561019f57600080fd5b506100bd610819565b670de0b6b3a764000081565b633b9aca0081565b602081565b6801bc16d674ec80000081565b6001600081815260209081527fada5013122d395ba3c54772283fb069b10426056ef8ca54750cb9bb552a59e7d805460408051600295831615610100026000190190921694909404601f8101849004840282018401909452838152606093909283018282801561027f5780601f106102545761010080835404028352916020019161027f565b820191906000526020600020905b81548152906001019060200180831161026257829003601f168201915b505050505090505b90565b600060208181529181526040908190208054825160026001831615610100026000190190921691909104601f81018590048502820185019093528281529290919083018282801561031c5780601f106102f15761010080835404028352916020019161031c565b820191906000526020600020905b8154815290600101906020018083116102ff57829003601f168201915b505050505081565b6201518081565b6001546401000000000160608080600080826103463461081f565b95506103514261081f565b94506103666103608787610850565b89610850565b93506000806001815260200190815260200160002060405180828054600181600116156101000203166002900480156103d65780601f106103b45761010080835404028352918201916103d6565b820191906000526020600020905b8154815290600101906020018083116103c2575b505091505060405180910390207fd60002d8f2bba463e7d533a8fa9b84d9d7451dc0bb7e9da868104601a5d2b9ef89600154604051610416929190610b22565b60405180910390a2836040518082805190602001908083835b6020831061044e5780518252601f19909201916020918201910161042f565b51815160209384036101000a60001901801990921691161790526040805192909401829003822082820152835180830382018152918401845260008d81528082529390932081516104a7965090945092019190506108c7565b50600092505b602083101561068d5760028704600281810260009081526020818152604091829020805483516001821615610100026000190190911694909404601f8101839004830285018301909352828452939a506105f99391908301828280156105545780601f1061052957610100808354040283529160200191610554565b820191906000526020600020905b81548152906001019060200180831161053757829003601f168201915b50505050506002898102600190810160009081526020818152604091829020805483519481161561010002600019011694909404601f81018290048202840182019092528183529192918301828280156105ef5780601f106105c4576101008083540402835291602001916105ef565b820191906000526020600020905b8154815290600101906020018083116105d257829003601f168201915b5050505050610850565b6040518082805190602001908083835b602083106106285780518252601f199092019160209182019101610609565b51815160209384036101000a60001901801990921691161790526040805192909401829003822082820152835180830382018152918401845260008d8152808252939093208151610681965090945092019190506108c7565b506001909201916104ad565b6801bc16d674ec80000034106106d8576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016106cf90610b58565b60405180910390fd5b670de0b6b3a76400003411610719576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016106cf90610b42565b6801bc16d674ec8000003414156107335760018054810190555b6140006001541415610809576201518080420642030191506107548261081f565b90506000806001815260200190815260200160002060405180828054600181600116156101000203166002900480156107c45780601f106107a25761010080835404028352918201916107c4565b820191906000526020600020905b8154815290600101906020018083116107b0575b505091505060405180910390207fb24eb1954aa12b3433be2b11740f2f73c8673453064575f93f223a85f1215f3a826040516108009190610b11565b60405180910390a25b5050505050505050565b61400081565b60015481565b6040805160208082528183019092526060918291908082016104008038833950505060208101939093525090919050565b815181516040518183018082526060939290916020601f8086018290049301049060005b8381101561089057600101602081028981015190830152610874565b5060005b828110156108b2576001016020810288810151908701830152610894565b50928301602001604052509095945050505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061090857805160ff1916838001178555610935565b82800160010185558215610935579182015b8281111561093557825182559160200191906001019061091a565b50610941929150610945565b5090565b61028791905b80821115610941576000815560010161094b565b6000601f8201831361097057600080fd5b813561098361097e82610b9d565b610b76565b9150808252602083016020830185838301111561099f57600080fd5b6109aa838284610bc9565b50505092915050565b60006109bf8235610287565b9392505050565b6000602082840312156109d857600080fd5b813567ffffffffffffffff8111156109ef57600080fd5b6109fb8482850161095f565b949350505050565b600060208284031215610a1557600080fd5b60006109fb84846109b3565b6000610a2c82610bc5565b808452610a40816020860160208601610bd5565b610a4981610c05565b9093016020019392505050565b602c81527f4465706f7369742063616e2774206265206c6573736572207468616e204d494e60208201527f5f544f5055505f53495a452e0000000000000000000000000000000000000000604082015260600190565b602b81527f4465706f7369742063616e27742062652067726561746572207468616e20444560208201527f504f5349545f53495a452e000000000000000000000000000000000000000000604082015260600190565b610b0b81610287565b82525050565b602080825281016109bf8184610a21565b60408082528101610b338185610a21565b90506109bf6020830184610b02565b60208082528101610b5281610a56565b92915050565b60208082528101610b5281610aac565b60208101610b528284610b02565b60405181810167ffffffffffffffff81118282101715610b9557600080fd5b604052919050565b600067ffffffffffffffff821115610bb457600080fd5b506020601f91909101601f19160190565b5190565b82818337506000910152565b60005b83811015610bf0578181015183820152602001610bd8565b83811115610bff576000848401525b50505050565b601f01601f1916905600a265627a7a7230582011335958f3420fb15ae87cd0b1cf2504b89aa41532651593ca4136531f686ff16c6578706572696d656e74616cf50037`

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
// Solidity: event ChainStart(receiptRoot indexed bytes, time bytes)
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
// Solidity: event ChainStart(receiptRoot indexed bytes, time bytes)
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
// Solidity: event HashChainValue(previousReceiptRoot indexed bytes, data bytes, totalDepositcount uint256)
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
// Solidity: event HashChainValue(previousReceiptRoot indexed bytes, data bytes, totalDepositcount uint256)
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
