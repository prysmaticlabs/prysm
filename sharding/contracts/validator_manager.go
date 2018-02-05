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
const VMCABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_expectedPeriodNumber\",\"type\":\"uint256\"},{\"name\":\"_periodStartPrevHash\",\"type\":\"bytes32\"},{\"name\":\"_parentCollationHash\",\"type\":\"bytes32\"},{\"name\":\"_txListRoot\",\"type\":\"bytes32\"},{\"name\":\"_collationCoinbase\",\"type\":\"address\"},{\"name\":\"_postStateRoot\",\"type\":\"bytes32\"},{\"name\":\"_receiptRoot\",\"type\":\"bytes32\"},{\"name\":\"_collationNumber\",\"type\":\"int256\"}],\"name\":\"addHeader\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"shardCount\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_txStartgas\",\"type\":\"uint256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes12\"}],\"name\":\"txToShard\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_validatorIndex\",\"type\":\"int256\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getCollationGasLimit\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"deposit\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_period\",\"type\":\"uint256\"}],\"name\":\"getEligibleProposer\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_receiptId\",\"type\":\"int256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"}],\"name\":\"updataGasPrice\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"shardId\",\"type\":\"int256\"},{\"indexed\":false,\"name\":\"receiptId\",\"type\":\"int256\"}],\"name\":\"TxToShard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"shardId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"collationHeader\",\"type\":\"bytes\"},{\"indexed\":false,\"name\":\"isNewHead\",\"type\":\"bool\"},{\"indexed\":false,\"name\":\"score\",\"type\":\"uint256\"}],\"name\":\"CollationAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"validator\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"index\",\"type\":\"int256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"validatorIndex\",\"type\":\"int256\"}],\"name\":\"Withdraw\",\"type\":\"event\"}]"

// VMCBin is the compiled bytecode used for deploying new contracts.
const VMCBin = `0x6060604052341561000f57600080fd5b6109448061001e6000396000f30060606040526004361061008d5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630341518d811461009257806304e9c77a146100de578063372a9e2a146101035780637e62eab81461012d578063934586ec14610145578063d0e30db014610158578063e29de3ad14610160578063e551e00a14610195575b600080fd5b341561009d57600080fd5b6100ca600435602435604435606435608435600160a060020a0360a4351660c43560e435610104356101a3565b604051901515815260200160405180910390f35b34156100e957600080fd5b6100f161040b565b60405190815260200160405180910390f35b6100f1600160a060020a0360043516602435604435606435600160a060020a031960843516610410565b341561013857600080fd5b610143600435610533565b005b341561015057600080fd5b6100f1610636565b6100f161063e565b341561016b57600080fd5b610179600435602435610763565b604051600160a060020a03909116815260200160405180910390f35b6100ca6004356024356107f1565b60006101ad6108f1565b60008b121580156101be575060648b125b15156101c957600080fd5b60054310156101d757600080fd5b600543048a146101e657600080fd5b60001960058b02014089146101fa57600080fd5b8a8a8a8a8a600160a060020a038b168a8a8a60405198895260208901979097526040808901969096526060880194909452608087019290925260a086015260c085015260e0840152610100830191909152610120909101905190819003902081528051151561026557fe5b60008b815260016020526040812090825181526020810191909152604001600020600101541561029157fe5b87156102c4578715806102bc575060008b81526001602081815260408084208c855290915282200154135b15156102c457fe5b60008b8152600960205260409020548a90126102dc57fe5b6102e98b60054304610763565b600160a060020a03166040820190815251600160a060020a0316151561030e57600080fd5b8060400151600160a060020a031633600160a060020a031614151561033257600080fd5b60008b81526001602081815260408084208c855282529092208101540190820190815251831461036157600080fd5b60408051908101604052888152602080820190830151905260008c81526001602052604081209083518152602081019190915260400160002081518155602082015160019182015560008d81526009602090815260408083208f905583825280832060038352818420548452825290912090910154915082015113156103fa57805160008c815260036020526040902055600160608201525b5060019a9950505050505050505050565b606481565b60008060e060405190810160409081528782526020808301889052818301879052346060840152600160a060020a031986166080840152600160a060020a0333811660a08501528a1660c08401526005546000908152600290915220815181556020820151816001015560408201518160020155606082015181600301556080820151600482015560a0820151600582018054600160a060020a031916600160a060020a039290921691909117905560c08201516006919091018054600160a060020a031916600160a060020a039283161790556005805460018101909155925087915088167ffc322e0c42ee41e0d74b940ceeee9cd5971acdd6ace8ff8010ee7134c31d9ea58360405190815260200160405180910390a39695505050505050565b60008181526020819052604090206001015433600160a060020a0390811691161461055d57600080fd5b6000818152602081905260409081902060018101549054600160a060020a039091169181156108fc02919051600060405180830381858888f1935050505015156105a657600080fd5b600081815260208181526040808320600181018054600160a060020a0316855260088452918420805460ff19169055848452918390529190558054600160a060020a03191690556105f681610836565b600480546000190190557fe13f360aa18d414ccdb598da6c447faa89d0477ffc7549dab5678fca76910b8c8160405190815260200160405180910390a150565b629896805b90565b600160a060020a033316600090815260086020526040812054819060ff161561066657600080fd5b3468056bc75e2d631000001461067b57600080fd5b610683610855565b15156106985761069161085c565b905061069d565b506004545b604080519081016040908152348252600160a060020a0333166020808401919091526000848152908190522081518155602082015160019182018054600160a060020a031916600160a060020a0392831617905560048054830190553390811660009081526008602052604090819020805460ff19169093179092557fd8a6d38df847dcba70dfdeb4948fb1457d61a81d132801f40dc9c00d52dfd478925090839051600160a060020a03909216825260208201526040908101905180910390a1919050565b6000600482101561077357600080fd5b4360031983016005021061078657600080fd5b6004546000901361079657600080fd5b6000806107a1610893565b600319850140600502866040519182526020820152604090810190519081900390208115156107cc57fe5b068152602081019190915260400160002060010154600160a060020a03169392505050565b60008281526002602052604081206005015433600160a060020a0390811691161461081b57600080fd5b50600091825260026020819052604090922090910155600190565b6007805460009081526006602052604090209190915580546001019055565b6007541590565b6000610866610855565b15610874575060001961063b565b5060078054600019019081905560009081526006602052604090205490565b600754600454600091829101815b6104008112156108e6578181126108b7576108e6565b600081815260208190526040902060010154600160a060020a0316156108de576001830192505b6001016108a1565b505060075401919050565b608060405190810160409081526000808352602083018190529082018190526060820152905600a165627a7a72305820cf6bda8a0b2efba379805dd0fa21d9896b2223809fe21406b302304d4ecb928f0029`

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
// Solidity: function addHeader(_shardId int256, _expectedPeriodNumber uint256, _periodStartPrevHash bytes32, _parentCollationHash bytes32, _txListRoot bytes32, _collationCoinbase address, _postStateRoot bytes32, _receiptRoot bytes32, _collationNumber int256) returns(bool)
func (_VMC *VMCTransactor) AddHeader(opts *bind.TransactOpts, _shardId *big.Int, _expectedPeriodNumber *big.Int, _periodStartPrevHash [32]byte, _parentCollationHash [32]byte, _txListRoot [32]byte, _collationCoinbase common.Address, _postStateRoot [32]byte, _receiptRoot [32]byte, _collationNumber *big.Int) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "addHeader", _shardId, _expectedPeriodNumber, _periodStartPrevHash, _parentCollationHash, _txListRoot, _collationCoinbase, _postStateRoot, _receiptRoot, _collationNumber)
}

// AddHeader is a paid mutator transaction binding the contract method 0x0341518d.
//
// Solidity: function addHeader(_shardId int256, _expectedPeriodNumber uint256, _periodStartPrevHash bytes32, _parentCollationHash bytes32, _txListRoot bytes32, _collationCoinbase address, _postStateRoot bytes32, _receiptRoot bytes32, _collationNumber int256) returns(bool)
func (_VMC *VMCSession) AddHeader(_shardId *big.Int, _expectedPeriodNumber *big.Int, _periodStartPrevHash [32]byte, _parentCollationHash [32]byte, _txListRoot [32]byte, _collationCoinbase common.Address, _postStateRoot [32]byte, _receiptRoot [32]byte, _collationNumber *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.AddHeader(&_VMC.TransactOpts, _shardId, _expectedPeriodNumber, _periodStartPrevHash, _parentCollationHash, _txListRoot, _collationCoinbase, _postStateRoot, _receiptRoot, _collationNumber)
}

// AddHeader is a paid mutator transaction binding the contract method 0x0341518d.
//
// Solidity: function addHeader(_shardId int256, _expectedPeriodNumber uint256, _periodStartPrevHash bytes32, _parentCollationHash bytes32, _txListRoot bytes32, _collationCoinbase address, _postStateRoot bytes32, _receiptRoot bytes32, _collationNumber int256) returns(bool)
func (_VMC *VMCTransactorSession) AddHeader(_shardId *big.Int, _expectedPeriodNumber *big.Int, _periodStartPrevHash [32]byte, _parentCollationHash [32]byte, _txListRoot [32]byte, _collationCoinbase common.Address, _postStateRoot [32]byte, _receiptRoot [32]byte, _collationNumber *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.AddHeader(&_VMC.TransactOpts, _shardId, _expectedPeriodNumber, _periodStartPrevHash, _parentCollationHash, _txListRoot, _collationCoinbase, _postStateRoot, _receiptRoot, _collationNumber)
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

// UpdataGasPrice is a paid mutator transaction binding the contract method 0xe551e00a.
//
// Solidity: function updataGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCTransactor) UpdataGasPrice(opts *bind.TransactOpts, _receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "updataGasPrice", _receiptId, _txGasprice)
}

// UpdataGasPrice is a paid mutator transaction binding the contract method 0xe551e00a.
//
// Solidity: function updataGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCSession) UpdataGasPrice(_receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.UpdataGasPrice(&_VMC.TransactOpts, _receiptId, _txGasprice)
}

// UpdataGasPrice is a paid mutator transaction binding the contract method 0xe551e00a.
//
// Solidity: function updataGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCTransactorSession) UpdataGasPrice(_receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.UpdataGasPrice(&_VMC.TransactOpts, _receiptId, _txGasprice)
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
