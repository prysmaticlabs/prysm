// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// VMCABI is the input ABI used to generate the binding from.
const VMCABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_expectedPeriodNumber\",\"type\":\"uint256\"},{\"name\":\"_periodStartPrevHash\",\"type\":\"bytes32\"},{\"name\":\"_parentHash\",\"type\":\"bytes32\"},{\"name\":\"_transactionRoot\",\"type\":\"bytes32\"},{\"name\":\"_coinbase\",\"type\":\"address\"},{\"name\":\"_stateRoot\",\"type\":\"bytes32\"},{\"name\":\"_receiptRoot\",\"type\":\"bytes32\"},{\"name\":\"_number\",\"type\":\"int256\"}],\"name\":\"addHeader\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"shardCount\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_receiptId\",\"type\":\"int256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"}],\"name\":\"updateGasPrice\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_txStartgas\",\"type\":\"uint256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes12\"}],\"name\":\"txToShard\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_validatorIndex\",\"type\":\"int256\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getCollationGasLimit\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"deposit\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_period\",\"type\":\"uint256\"}],\"name\":\"getEligibleProposer\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"shardId\",\"type\":\"int256\"},{\"indexed\":false,\"name\":\"receiptId\",\"type\":\"int256\"}],\"name\":\"TxToShard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"shardId\",\"type\":\"int256\"},{\"indexed\":false,\"name\":\"expectedPeriodNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"periodStartPrevHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"parentHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"transactionRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"coinbase\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"receiptRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"number\",\"type\":\"int256\"},{\"indexed\":false,\"name\":\"isNewHead\",\"type\":\"bool\"},{\"indexed\":false,\"name\":\"score\",\"type\":\"int256\"}],\"name\":\"CollationAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"index\",\"type\":\"int256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"validatorIndex\",\"type\":\"int256\"}],\"name\":\"Withdraw\",\"type\":\"event\"}]"

// VMCBin is the compiled bytecode used for deploying new contracts.
const VMCBin = `0x6060604052341561000f57600080fd5b6109c28061001e6000396000f30060606040526004361061008d5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630341518d811461009257806304e9c77a146100de5780632213138914610103578063372a9e2a146101115780637e62eab81461013b578063934586ec14610153578063d0e30db014610166578063e29de3ad1461016e575b600080fd5b341561009d57600080fd5b6100ca600435602435604435606435608435600160a060020a0360a4351660c43560e435610104356101a3565b604051901515815260200160405180910390f35b34156100e957600080fd5b6100f1610489565b60405190815260200160405180910390f35b6100ca60043560243561048e565b6100f1600160a060020a0360043516602435604435606435600160a060020a0319608435166104d3565b341561014657600080fd5b6101516004356105f6565b005b341561015e57600080fd5b6100f16106f9565b6100f1610701565b341561017957600080fd5b610187600435602435610826565b604051600160a060020a03909116815260200160405180910390f35b60006101ad61096f565b60008b121580156101be575060648b125b15156101c957600080fd5b60054310156101d757600080fd5b600543048a146101e657600080fd5b60001960058b02014089146101fa57600080fd5b8a8a8a8a8a600160a060020a038b168a8a8a60405198895260208901979097526040808901969096526060880194909452608087019290925260a086015260c085015260e08401526101008301919091526101209091019051908190039020815260008b815260016020526040812090825181526020810191909152604001600020600101541561028757fe5b87156102af5760008b81526001602081815260408084208c855290915282200154136102af57fe5b60008b8152600960205260409020548a90126102c757fe5b6102d48b60054304610826565b600160a060020a03166040820190815251600160a060020a031615156102f957600080fd5b8060400151600160a060020a031633600160a060020a031614151561031d57600080fd5b60008b81526001602081815260408084208c855282529092208101540190820190815251831461034c57600080fd5b60408051908101604052888152602080820190830151905260008c81526001602052604081209083518152602081019190915260400160002081518155602082015160019182015560008d81526009602090815260408083208f905583825280832060038352818420548452825290912090910154915082015113156103e557805160008c815260036020526040902055600160608201525b8a7f788a01fdbeb989d24e706cdd2993ca4213400e7b5fa631819b2acfe8ebad41358b8b8b8b8b8b8b8b8a606001518b60200151604051998a5260208a01989098526040808a01979097526060890195909552600160a060020a03909316608088015260a087019190915260c086015260e08501521515610100840152610120830191909152610140909101905180910390a25060019a9950505050505050505050565b606481565b60008281526002602052604081206005015433600160a060020a039081169116146104b857600080fd5b50600091825260026020819052604090922090910155600190565b60008060e060405190810160409081528782526020808301889052818301879052346060840152600160a060020a031986166080840152600160a060020a0333811660a08501528a1660c08401526005546000908152600290915220815181556020820151816001015560408201518160020155606082015181600301556080820151600482015560a0820151600582018054600160a060020a031916600160a060020a039290921691909117905560c08201516006919091018054600160a060020a031916600160a060020a039283161790556005805460018101909155925087915088167ffc322e0c42ee41e0d74b940ceeee9cd5971acdd6ace8ff8010ee7134c31d9ea58360405190815260200160405180910390a39695505050505050565b60008181526020819052604090206001015433600160a060020a0390811691161461062057600080fd5b6000818152602081905260409081902060018101549054600160a060020a039091169181156108fc02919051600060405180830381858888f19350505050151561066957600080fd5b600081815260208181526040808320600181018054600160a060020a0316855260088452918420805460ff19169055848452918390529190558054600160a060020a03191690556106b9816108b4565b600480546000190190557fe13f360aa18d414ccdb598da6c447faa89d0477ffc7549dab5678fca76910b8c8160405190815260200160405180910390a150565b629896805b90565b600160a060020a033316600090815260086020526040812054819060ff161561072957600080fd5b3468056bc75e2d631000001461073e57600080fd5b6107466108d3565b151561075b576107546108da565b9050610760565b506004545b604080519081016040908152348252600160a060020a0333166020808401919091526000848152908190522081518155602082015160019182018054600160a060020a031916600160a060020a0392831617905560048054830190553390811660009081526008602052604090819020805460ff19169093179092557fd8a6d38df847dcba70dfdeb4948fb1457d61a81d132801f40dc9c00d52dfd478925090839051600160a060020a03909216825260208201526040908101905180910390a1919050565b6000600482101561083657600080fd5b4360031983016005021061084957600080fd5b6004546000901361085957600080fd5b600080610864610911565b6003198501406005028660405191825260208201526040908101905190819003902081151561088f57fe5b068152602081019190915260400160002060010154600160a060020a03169392505050565b6007805460009081526006602052604090209190915580546001019055565b6007541590565b60006108e46108d3565b156108f257506000196106fe565b5060078054600019019081905560009081526006602052604090205490565b600754600454600091829101815b6104008112156109645781811261093557610964565b600081815260208190526040902060010154600160a060020a03161561095c576001830192505b60010161091f565b505060075401919050565b608060405190810160409081526000808352602083018190529082018190526060820152905600a165627a7a72305820ddfce668015ac34d63ee4a37f827d5153b83879b7d76b675b9d34bb79742dff90029`

// DeployVMC deploys a new Ethereum contract, binding an instance of VMC to it.
func DeployVMC(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *VMC, error) {
	parsed, err := abi.JSON(strings.NewReader(VMCABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(VMCBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &VMC{VMCCaller: VMCCaller{contract: contract}, VMCTransactor: VMCTransactor{contract: contract}}, nil
}

// VMC is an auto generated Go binding around an Ethereum contract.
type VMC struct {
	VMCCaller     // Read-only binding to the contract
	VMCTransactor // Write-only binding to the contract
}

// VMCCaller is an auto generated read-only Go binding around an Ethereum contract.
type VMCCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VMCTransactor is an auto generated write-only Go binding around an Ethereum contract.
type VMCTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VMCSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type VMCSession struct {
	Contract     *VMC              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VMCCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type VMCCallerSession struct {
	Contract *VMCCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// VMCTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type VMCTransactorSession struct {
	Contract     *VMCTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VMCRaw is an auto generated low-level Go binding around an Ethereum contract.
type VMCRaw struct {
	Contract *VMC // Generic contract binding to access the raw methods on
}

// VMCCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type VMCCallerRaw struct {
	Contract *VMCCaller // Generic read-only contract binding to access the raw methods on
}

// VMCTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type VMCTransactorRaw struct {
	Contract *VMCTransactor // Generic write-only contract binding to access the raw methods on
}

// NewVMC creates a new instance of VMC, bound to a specific deployed contract.
func NewVMC(address common.Address, backend bind.ContractBackend) (*VMC, error) {
	contract, err := bindVMC(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &VMC{VMCCaller: VMCCaller{contract: contract}, VMCTransactor: VMCTransactor{contract: contract}}, nil
}

// NewVMCCaller creates a new read-only instance of VMC, bound to a specific deployed contract.
func NewVMCCaller(address common.Address, caller bind.ContractCaller) (*VMCCaller, error) {
	contract, err := bindVMC(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &VMCCaller{contract: contract}, nil
}

// NewVMCTransactor creates a new write-only instance of VMC, bound to a specific deployed contract.
func NewVMCTransactor(address common.Address, transactor bind.ContractTransactor) (*VMCTransactor, error) {
	contract, err := bindVMC(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &VMCTransactor{contract: contract}, nil
}

// bindVMC binds a generic wrapper to an already deployed contract.
func bindVMC(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(VMCABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VMC *VMCRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _VMC.Contract.VMCCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VMC *VMCRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VMC.Contract.VMCTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VMC *VMCRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VMC.Contract.VMCTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VMC *VMCCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _VMC.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VMC *VMCTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VMC.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VMC *VMCTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VMC.Contract.contract.Transact(opts, method, params...)
}

// GetCollationGasLimit is a free data retrieval call binding the contract method 0x934586ec.
//
// Solidity: function getCollationGasLimit() constant returns(uint256)
func (_VMC *VMCCaller) GetCollationGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getCollationGasLimit")
	return *ret0, err
}

// GetCollationGasLimit is a free data retrieval call binding the contract method 0x934586ec.
//
// Solidity: function getCollationGasLimit() constant returns(uint256)
func (_VMC *VMCSession) GetCollationGasLimit() (*big.Int, error) {
	return _VMC.Contract.GetCollationGasLimit(&_VMC.CallOpts)
}

// GetCollationGasLimit is a free data retrieval call binding the contract method 0x934586ec.
//
// Solidity: function getCollationGasLimit() constant returns(uint256)
func (_VMC *VMCCallerSession) GetCollationGasLimit() (*big.Int, error) {
	return _VMC.Contract.GetCollationGasLimit(&_VMC.CallOpts)
}

// GetEligibleProposer is a free data retrieval call binding the contract method 0xe29de3ad.
//
// Solidity: function getEligibleProposer(_shardId int256, _period uint256) constant returns(address)
func (_VMC *VMCCaller) GetEligibleProposer(opts *bind.CallOpts, _shardId *big.Int, _period *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getEligibleProposer", _shardId, _period)
	return *ret0, err
}

// GetEligibleProposer is a free data retrieval call binding the contract method 0xe29de3ad.
//
// Solidity: function getEligibleProposer(_shardId int256, _period uint256) constant returns(address)
func (_VMC *VMCSession) GetEligibleProposer(_shardId *big.Int, _period *big.Int) (common.Address, error) {
	return _VMC.Contract.GetEligibleProposer(&_VMC.CallOpts, _shardId, _period)
}

// GetEligibleProposer is a free data retrieval call binding the contract method 0xe29de3ad.
//
// Solidity: function getEligibleProposer(_shardId int256, _period uint256) constant returns(address)
func (_VMC *VMCCallerSession) GetEligibleProposer(_shardId *big.Int, _period *big.Int) (common.Address, error) {
	return _VMC.Contract.GetEligibleProposer(&_VMC.CallOpts, _shardId, _period)
}

// ShardCount is a free data retrieval call binding the contract method 0x04e9c77a.
//
// Solidity: function shardCount() constant returns(int256)
func (_VMC *VMCCaller) ShardCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "shardCount")
	return *ret0, err
}

// ShardCount is a free data retrieval call binding the contract method 0x04e9c77a.
//
// Solidity: function shardCount() constant returns(int256)
func (_VMC *VMCSession) ShardCount() (*big.Int, error) {
	return _VMC.Contract.ShardCount(&_VMC.CallOpts)
}

// ShardCount is a free data retrieval call binding the contract method 0x04e9c77a.
//
// Solidity: function shardCount() constant returns(int256)
func (_VMC *VMCCallerSession) ShardCount() (*big.Int, error) {
	return _VMC.Contract.ShardCount(&_VMC.CallOpts)
}

// AddHeader is a paid mutator transaction binding the contract method 0x0341518d.
//
// Solidity: function addHeader(_shardId int256, _expectedPeriodNumber uint256, _periodStartPrevHash bytes32, _parentHash bytes32, _transactionRoot bytes32, _coinbase address, _stateRoot bytes32, _receiptRoot bytes32, _number int256) returns(bool)
func (_VMC *VMCTransactor) AddHeader(opts *bind.TransactOpts, _shardId *big.Int, _expectedPeriodNumber *big.Int, _periodStartPrevHash [32]byte, _parentHash [32]byte, _transactionRoot [32]byte, _coinbase common.Address, _stateRoot [32]byte, _receiptRoot [32]byte, _number *big.Int) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "addHeader", _shardId, _expectedPeriodNumber, _periodStartPrevHash, _parentHash, _transactionRoot, _coinbase, _stateRoot, _receiptRoot, _number)
}

// AddHeader is a paid mutator transaction binding the contract method 0x0341518d.
//
// Solidity: function addHeader(_shardId int256, _expectedPeriodNumber uint256, _periodStartPrevHash bytes32, _parentHash bytes32, _transactionRoot bytes32, _coinbase address, _stateRoot bytes32, _receiptRoot bytes32, _number int256) returns(bool)
func (_VMC *VMCSession) AddHeader(_shardId *big.Int, _expectedPeriodNumber *big.Int, _periodStartPrevHash [32]byte, _parentHash [32]byte, _transactionRoot [32]byte, _coinbase common.Address, _stateRoot [32]byte, _receiptRoot [32]byte, _number *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.AddHeader(&_VMC.TransactOpts, _shardId, _expectedPeriodNumber, _periodStartPrevHash, _parentHash, _transactionRoot, _coinbase, _stateRoot, _receiptRoot, _number)
}

// AddHeader is a paid mutator transaction binding the contract method 0x0341518d.
//
// Solidity: function addHeader(_shardId int256, _expectedPeriodNumber uint256, _periodStartPrevHash bytes32, _parentHash bytes32, _transactionRoot bytes32, _coinbase address, _stateRoot bytes32, _receiptRoot bytes32, _number int256) returns(bool)
func (_VMC *VMCTransactorSession) AddHeader(_shardId *big.Int, _expectedPeriodNumber *big.Int, _periodStartPrevHash [32]byte, _parentHash [32]byte, _transactionRoot [32]byte, _coinbase common.Address, _stateRoot [32]byte, _receiptRoot [32]byte, _number *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.AddHeader(&_VMC.TransactOpts, _shardId, _expectedPeriodNumber, _periodStartPrevHash, _parentHash, _transactionRoot, _coinbase, _stateRoot, _receiptRoot, _number)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() returns(int256)
func (_VMC *VMCTransactor) Deposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "deposit")
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() returns(int256)
func (_VMC *VMCSession) Deposit() (*types.Transaction, error) {
	return _VMC.Contract.Deposit(&_VMC.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() returns(int256)
func (_VMC *VMCTransactorSession) Deposit() (*types.Transaction, error) {
	return _VMC.Contract.Deposit(&_VMC.TransactOpts)
}

// TxToShard is a paid mutator transaction binding the contract method 0x372a9e2a.
//
// Solidity: function txToShard(_to address, _shardId int256, _txStartgas uint256, _txGasprice uint256, _data bytes12) returns(int256)
func (_VMC *VMCTransactor) TxToShard(opts *bind.TransactOpts, _to common.Address, _shardId *big.Int, _txStartgas *big.Int, _txGasprice *big.Int, _data [12]byte) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "txToShard", _to, _shardId, _txStartgas, _txGasprice, _data)
}

// TxToShard is a paid mutator transaction binding the contract method 0x372a9e2a.
//
// Solidity: function txToShard(_to address, _shardId int256, _txStartgas uint256, _txGasprice uint256, _data bytes12) returns(int256)
func (_VMC *VMCSession) TxToShard(_to common.Address, _shardId *big.Int, _txStartgas *big.Int, _txGasprice *big.Int, _data [12]byte) (*types.Transaction, error) {
	return _VMC.Contract.TxToShard(&_VMC.TransactOpts, _to, _shardId, _txStartgas, _txGasprice, _data)
}

// TxToShard is a paid mutator transaction binding the contract method 0x372a9e2a.
//
// Solidity: function txToShard(_to address, _shardId int256, _txStartgas uint256, _txGasprice uint256, _data bytes12) returns(int256)
func (_VMC *VMCTransactorSession) TxToShard(_to common.Address, _shardId *big.Int, _txStartgas *big.Int, _txGasprice *big.Int, _data [12]byte) (*types.Transaction, error) {
	return _VMC.Contract.TxToShard(&_VMC.TransactOpts, _to, _shardId, _txStartgas, _txGasprice, _data)
}

// UpdateGasPrice is a paid mutator transaction binding the contract method 0x22131389.
//
// Solidity: function updateGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCTransactor) UpdateGasPrice(opts *bind.TransactOpts, _receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "updateGasPrice", _receiptId, _txGasprice)
}

// UpdateGasPrice is a paid mutator transaction binding the contract method 0x22131389.
//
// Solidity: function updateGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCSession) UpdateGasPrice(_receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.UpdateGasPrice(&_VMC.TransactOpts, _receiptId, _txGasprice)
}

// UpdateGasPrice is a paid mutator transaction binding the contract method 0x22131389.
//
// Solidity: function updateGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCTransactorSession) UpdateGasPrice(_receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.UpdateGasPrice(&_VMC.TransactOpts, _receiptId, _txGasprice)
}

// Withdraw is a paid mutator transaction binding the contract method 0x7e62eab8.
//
// Solidity: function withdraw(_validatorIndex int256) returns()
func (_VMC *VMCTransactor) Withdraw(opts *bind.TransactOpts, _validatorIndex *big.Int) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "withdraw", _validatorIndex)
}

// Withdraw is a paid mutator transaction binding the contract method 0x7e62eab8.
//
// Solidity: function withdraw(_validatorIndex int256) returns()
func (_VMC *VMCSession) Withdraw(_validatorIndex *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.Withdraw(&_VMC.TransactOpts, _validatorIndex)
}

// Withdraw is a paid mutator transaction binding the contract method 0x7e62eab8.
//
// Solidity: function withdraw(_validatorIndex int256) returns()
func (_VMC *VMCTransactorSession) Withdraw(_validatorIndex *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.Withdraw(&_VMC.TransactOpts, _validatorIndex)
}
