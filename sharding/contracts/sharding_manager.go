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

// SMCABI is the input ABI used to generate the binding from.
const SMCABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"name\":\"period_head\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"register_proposer\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"release_proposer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"shard_id\",\"type\":\"uint256\"}],\"name\":\"proposer_add_balance\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"},{\"name\":\"period\",\"type\":\"uint256\"},{\"name\":\"height\",\"type\":\"bytes32\"},{\"name\":\"_parent_hash\",\"type\":\"bytes32\"},{\"name\":\"chunk_root\",\"type\":\"bytes32\"},{\"name\":\"proposer_address\",\"type\":\"address\"},{\"name\":\"proposer_bid\",\"type\":\"uint256\"},{\"name\":\"proposer_signature\",\"type\":\"bytes\"}],\"name\":\"addHeader\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"proposer_pool\",\"outputs\":[{\"name\":\"shardId\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"shard_id\",\"type\":\"uint256\"},{\"name\":\"parent_hash\",\"type\":\"bytes32\"},{\"name\":\"chunk_root\",\"type\":\"bytes32\"},{\"name\":\"period\",\"type\":\"uint256\"},{\"name\":\"proposer_address\",\"type\":\"address\"},{\"name\":\"proposer_bid\",\"type\":\"uint256\"}],\"name\":\"compute_header_hash\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"collator_pool_len\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"proposer_registry\",\"outputs\":[{\"name\":\"shardId\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"shard_id\",\"type\":\"uint256\"}],\"name\":\"proposer_withdraw_balance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"collator_pool\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"int256\"},{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"collationHeaders\",\"outputs\":[{\"name\":\"shard_id\",\"type\":\"uint256\"},{\"name\":\"parent_hash\",\"type\":\"bytes32\"},{\"name\":\"chunk_root\",\"type\":\"bytes32\"},{\"name\":\"period\",\"type\":\"int128\"},{\"name\":\"height\",\"type\":\"int128\"},{\"name\":\"proposer_address\",\"type\":\"address\"},{\"name\":\"proposer_bid\",\"type\":\"uint256\"},{\"name\":\"proposer_signature\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"collator_registry\",\"outputs\":[{\"name\":\"deregistered\",\"type\":\"uint256\"},{\"name\":\"pool_index\",\"type\":\"uint256\"},{\"name\":\"deposited\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"register_collator\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"release_collator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"collation_trees\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"shard_id\",\"type\":\"uint256\"},{\"name\":\"period\",\"type\":\"uint256\"}],\"name\":\"get_eligible_collator\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"deregister_collator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"deregister_proposer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"shard_id\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"parent_hash\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"chunk_root\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"period\",\"type\":\"int128\"},{\"indexed\":false,\"name\":\"height\",\"type\":\"int128\"},{\"indexed\":false,\"name\":\"proposer_address\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"proposer_bid\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"proposer_signature\",\"type\":\"bytes\"}],\"name\":\"HeaderAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"collator\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"pool_index\",\"type\":\"uint256\"}],\"name\":\"CollatorRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"pool_index\",\"type\":\"uint256\"}],\"name\":\"CollatorDeregistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"pool_index\",\"type\":\"uint256\"}],\"name\":\"CollatorReleased\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"pool_index\",\"type\":\"uint256\"}],\"name\":\"ProposerRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"ProposerDeregistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"ProposerReleased\",\"type\":\"event\"}]"

// SMCBin is the compiled bytecode used for deploying new contracts.
const SMCBin = `0x6060604052341561000f57600080fd5b610a598061001e6000396000f3006060604052600436106101065763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416634462ffae811461010b578063531b6fc7146101335780635950d0ca1461013b5780635f7d96b514610150578063683eee781461015b5780636d9061c3146101e85780636e6483f4146101fe57806394c30f231461022c57806398bd94df1461023f5780639c56a7e01461025e578063ade8d06f14610269578063b9d8ef961461029b578063cc05589c14610383578063d64e1650146103c8578063d763dfd0146103d0578063ea2ef69e146103e3578063f3ae61b7146103fc578063fae39b4f14610415578063ff6831391461013b575b600080fd5b341561011657600080fd5b610121600435610428565b60405190815260200160405180910390f35b61012161043a565b341561014657600080fd5b61014e61043f565b005b61014e600435610441565b341561016657600080fd5b6101d460048035906024803591604435916064359160843591600160a060020a0360a435169160c435916101049060e43590810190830135806020601f8201819004810201604051908101604052818152929190602084018383808284375094965061044495505050505050565b604051901515815260200160405180910390f35b34156101f357600080fd5b610121600435610452565b341561020957600080fd5b610121600435602435604435606435600160a060020a036084351660a435610476565b341561023757600080fd5b610121610482565b341561024a57600080fd5b610121600160a060020a0360043516610488565b341561015057600080fd5b341561027457600080fd5b61027f60043561049a565b604051600160a060020a03909116815260200160405180910390f35b34156102a657600080fd5b6102b46004356024356104c2565b6040518881526020810188905260408101879052600f86810b810b606083015285810b900b6080820152600160a060020a03841660a082015260c0810183905261010060e0820181815283546002600182161584026000190190911604918301829052906101208301908490801561036d5780601f106103425761010080835404028352916020019161036d565b820191906000526020600020905b81548152906001019060200180831161035057829003601f168201915b5050995050505050505050505060405180910390f35b341561038e57600080fd5b6103a2600160a060020a036004351661052b565b604051928352602083019190915215156040808301919091526060909101905180910390f35b6101d461054e565b34156103db57600080fd5b61014e6106ee565b34156103ee57600080fd5b6101216004356024356107c6565b341561040757600080fd5b61027f6004356024356107e3565b341561042057600080fd5b61014e610870565b600a6020526000908152604090205481565b600090565b565b50565b600098975050505050505050565b600180548290811061046057fe5b6000918252602090912060029091020154905081565b60009695505050505050565b60075481565b60036020526000908152604090205481565b60008054829081106104a857fe5b600091825260209091200154600160a060020a0316905081565b600560208181526000938452604080852090915291835291208054600182015460028301546003840154600485015495850154939592949193600f82810b94700100000000000000000000000000000000909304900b92600160a060020a039092169160060188565b600260208190526000918252604090912080546001820154919092015460ff1683565b33600160a060020a038116600090815260026020819052604082200154909190829060ff161561057d57600080fd5b34683635c9adc5dea000001461059257600080fd5b61059a610942565b15156105f5576105a861094a565b9050816000828154811015156105ba57fe5b6000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a039290921691909117905561063e565b50600754600080546001810161060b83826109e6565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0384161790555b6007805460010190556060604051908101604090815260008083526020808401859052600183850152600160a060020a0386168252600290522081518155602082015181600101556040820151600291909101805460ff1916911515919091179055507fb3cf9a630836c5786f2ca8e1c1db9b946b76f3f7f18aa30514214a9ad16ecfc38282604051600160a060020a03909216825260208201526040908101905180910390a160019250505090565b600160a060020a03339081166000908152600260208190526040909120015460ff16151560011461071e57600080fd5b600160a060020a038116600090815260026020526040902054151561074257600080fd5b600160a060020a038116600090815260026020526040902054613f0001600543041161076d57600080fd5b600160a060020a038116600081815260026020819052604080832083815560018101849055909101805460ff19169055683635c9adc5dea000009051600060405180830381858888f19350505050151561044157600080fd5b600460209081526000928352604080842090915290825290205481565b600060048210156107f357600080fd5b4360031983016005021061080657600080fd5b6007546000901161081657600080fd5b6007546000906003198401600502408560405191825260208201526040908101905190819003902081151561084757fe5b0681548110151561085457fe5b600091825260209091200154600160a060020a03169392505050565b33600160a060020a0381166000908152600260208190526040909120600181015491015460ff1615156108a257600080fd5b81600160a060020a03166000828154811015156108bb57fe5b600091825260209091200154600160a060020a0316146108da57600080fd5b600160a060020a038216600090815260026020526040902060054304905561090181610989565b600780546000190190557f9b2d06607fbfb01da25cdaa6312dfbc880589de849cdb80dae0a0e8c87e097c68160405190815260200160405180910390a15050565b600954155b90565b6000600160095411151561095d57600080fd5b60098054600019019081905560088054909190811061097857fe5b906000526020600020900154905090565b60095460085414156109bb5760088054600181016109a783826109e6565b5060009182526020909120018190556109da565b8060086009548154811015156109cd57fe5b6000918252602090912001555b50600980546001019055565b815481835581811511610a0a57600083815260209020610a0a918101908301610a0f565b505050565b61094791905b80821115610a295760008155600101610a15565b50905600a165627a7a72305820d4812a21ffa1a13d4818a30fbd6189b681d08a7b6dccf2f03d7f97f0bdb5d5e60029`

// DeploySMC deploys a new Ethereum contract, binding an instance of SMC to it.
func DeploySMC(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SMC, error) {
	parsed, err := abi.JSON(strings.NewReader(SMCABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SMCBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SMC{SMCCaller: SMCCaller{contract: contract}, SMCTransactor: SMCTransactor{contract: contract}, SMCFilterer: SMCFilterer{contract: contract}}, nil
}

// SMC is an auto generated Go binding around an Ethereum contract.
type SMC struct {
	SMCCaller     // Read-only binding to the contract
	SMCTransactor // Write-only binding to the contract
	SMCFilterer   // Log filterer for contract events
}

// SMCCaller is an auto generated read-only Go binding around an Ethereum contract.
type SMCCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SMCTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SMCTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SMCFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SMCFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SMCSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SMCSession struct {
	Contract     *SMC              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SMCCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SMCCallerSession struct {
	Contract *SMCCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// SMCTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SMCTransactorSession struct {
	Contract     *SMCTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SMCRaw is an auto generated low-level Go binding around an Ethereum contract.
type SMCRaw struct {
	Contract *SMC // Generic contract binding to access the raw methods on
}

// SMCCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SMCCallerRaw struct {
	Contract *SMCCaller // Generic read-only contract binding to access the raw methods on
}

// SMCTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SMCTransactorRaw struct {
	Contract *SMCTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSMC creates a new instance of SMC, bound to a specific deployed contract.
func NewSMC(address common.Address, backend bind.ContractBackend) (*SMC, error) {
	contract, err := bindSMC(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SMC{SMCCaller: SMCCaller{contract: contract}, SMCTransactor: SMCTransactor{contract: contract}, SMCFilterer: SMCFilterer{contract: contract}}, nil
}

// NewSMCCaller creates a new read-only instance of SMC, bound to a specific deployed contract.
func NewSMCCaller(address common.Address, caller bind.ContractCaller) (*SMCCaller, error) {
	contract, err := bindSMC(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SMCCaller{contract: contract}, nil
}

// NewSMCTransactor creates a new write-only instance of SMC, bound to a specific deployed contract.
func NewSMCTransactor(address common.Address, transactor bind.ContractTransactor) (*SMCTransactor, error) {
	contract, err := bindSMC(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SMCTransactor{contract: contract}, nil
}

// NewSMCFilterer creates a new log filterer instance of SMC, bound to a specific deployed contract.
func NewSMCFilterer(address common.Address, filterer bind.ContractFilterer) (*SMCFilterer, error) {
	contract, err := bindSMC(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SMCFilterer{contract: contract}, nil
}

// bindSMC binds a generic wrapper to an already deployed contract.
func bindSMC(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SMCABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SMC *SMCRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SMC.Contract.SMCCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SMC *SMCRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.Contract.SMCTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SMC *SMCRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SMC.Contract.SMCTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SMC *SMCCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SMC.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SMC *SMCTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SMC *SMCTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SMC.Contract.contract.Transact(opts, method, params...)
}

// CollationHeaders is a free data retrieval call binding the contract method 0xb9d8ef96.
//
// Solidity: function collationHeaders( int256,  bytes32) constant returns(shard_id uint256, parent_hash bytes32, chunk_root bytes32, period int128, height int128, proposer_address address, proposer_bid uint256, proposer_signature bytes)
func (_SMC *SMCCaller) CollationHeaders(opts *bind.CallOpts, arg0 *big.Int, arg1 [32]byte) (struct {
	Shard_id           *big.Int
	Parent_hash        [32]byte
	Chunk_root         [32]byte
	Period             *big.Int
	Height             *big.Int
	Proposer_address   common.Address
	Proposer_bid       *big.Int
	Proposer_signature []byte
}, error) {
	ret := new(struct {
		Shard_id           *big.Int
		Parent_hash        [32]byte
		Chunk_root         [32]byte
		Period             *big.Int
		Height             *big.Int
		Proposer_address   common.Address
		Proposer_bid       *big.Int
		Proposer_signature []byte
	})
	out := ret
	err := _SMC.contract.Call(opts, out, "collationHeaders", arg0, arg1)
	return *ret, err
}

// CollationHeaders is a free data retrieval call binding the contract method 0xb9d8ef96.
//
// Solidity: function collationHeaders( int256,  bytes32) constant returns(shard_id uint256, parent_hash bytes32, chunk_root bytes32, period int128, height int128, proposer_address address, proposer_bid uint256, proposer_signature bytes)
func (_SMC *SMCSession) CollationHeaders(arg0 *big.Int, arg1 [32]byte) (struct {
	Shard_id           *big.Int
	Parent_hash        [32]byte
	Chunk_root         [32]byte
	Period             *big.Int
	Height             *big.Int
	Proposer_address   common.Address
	Proposer_bid       *big.Int
	Proposer_signature []byte
}, error) {
	return _SMC.Contract.CollationHeaders(&_SMC.CallOpts, arg0, arg1)
}

// CollationHeaders is a free data retrieval call binding the contract method 0xb9d8ef96.
//
// Solidity: function collationHeaders( int256,  bytes32) constant returns(shard_id uint256, parent_hash bytes32, chunk_root bytes32, period int128, height int128, proposer_address address, proposer_bid uint256, proposer_signature bytes)
func (_SMC *SMCCallerSession) CollationHeaders(arg0 *big.Int, arg1 [32]byte) (struct {
	Shard_id           *big.Int
	Parent_hash        [32]byte
	Chunk_root         [32]byte
	Period             *big.Int
	Height             *big.Int
	Proposer_address   common.Address
	Proposer_bid       *big.Int
	Proposer_signature []byte
}, error) {
	return _SMC.Contract.CollationHeaders(&_SMC.CallOpts, arg0, arg1)
}

// Collation_trees is a free data retrieval call binding the contract method 0xea2ef69e.
//
// Solidity: function collation_trees( uint256,  bytes32) constant returns(bytes32)
func (_SMC *SMCCaller) Collation_trees(opts *bind.CallOpts, arg0 *big.Int, arg1 [32]byte) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "collation_trees", arg0, arg1)
	return *ret0, err
}

// Collation_trees is a free data retrieval call binding the contract method 0xea2ef69e.
//
// Solidity: function collation_trees( uint256,  bytes32) constant returns(bytes32)
func (_SMC *SMCSession) Collation_trees(arg0 *big.Int, arg1 [32]byte) ([32]byte, error) {
	return _SMC.Contract.Collation_trees(&_SMC.CallOpts, arg0, arg1)
}

// Collation_trees is a free data retrieval call binding the contract method 0xea2ef69e.
//
// Solidity: function collation_trees( uint256,  bytes32) constant returns(bytes32)
func (_SMC *SMCCallerSession) Collation_trees(arg0 *big.Int, arg1 [32]byte) ([32]byte, error) {
	return _SMC.Contract.Collation_trees(&_SMC.CallOpts, arg0, arg1)
}

// Collator_pool is a free data retrieval call binding the contract method 0xade8d06f.
//
// Solidity: function collator_pool( uint256) constant returns(address)
func (_SMC *SMCCaller) Collator_pool(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "collator_pool", arg0)
	return *ret0, err
}

// Collator_pool is a free data retrieval call binding the contract method 0xade8d06f.
//
// Solidity: function collator_pool( uint256) constant returns(address)
func (_SMC *SMCSession) Collator_pool(arg0 *big.Int) (common.Address, error) {
	return _SMC.Contract.Collator_pool(&_SMC.CallOpts, arg0)
}

// Collator_pool is a free data retrieval call binding the contract method 0xade8d06f.
//
// Solidity: function collator_pool( uint256) constant returns(address)
func (_SMC *SMCCallerSession) Collator_pool(arg0 *big.Int) (common.Address, error) {
	return _SMC.Contract.Collator_pool(&_SMC.CallOpts, arg0)
}

// Collator_pool_len is a free data retrieval call binding the contract method 0x94c30f23.
//
// Solidity: function collator_pool_len() constant returns(uint256)
func (_SMC *SMCCaller) Collator_pool_len(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "collator_pool_len")
	return *ret0, err
}

// Collator_pool_len is a free data retrieval call binding the contract method 0x94c30f23.
//
// Solidity: function collator_pool_len() constant returns(uint256)
func (_SMC *SMCSession) Collator_pool_len() (*big.Int, error) {
	return _SMC.Contract.Collator_pool_len(&_SMC.CallOpts)
}

// Collator_pool_len is a free data retrieval call binding the contract method 0x94c30f23.
//
// Solidity: function collator_pool_len() constant returns(uint256)
func (_SMC *SMCCallerSession) Collator_pool_len() (*big.Int, error) {
	return _SMC.Contract.Collator_pool_len(&_SMC.CallOpts)
}

// Collator_registry is a free data retrieval call binding the contract method 0xcc05589c.
//
// Solidity: function collator_registry( address) constant returns(deregistered uint256, pool_index uint256, deposited bool)
func (_SMC *SMCCaller) Collator_registry(opts *bind.CallOpts, arg0 common.Address) (struct {
	Deregistered *big.Int
	Pool_index   *big.Int
	Deposited    bool
}, error) {
	ret := new(struct {
		Deregistered *big.Int
		Pool_index   *big.Int
		Deposited    bool
	})
	out := ret
	err := _SMC.contract.Call(opts, out, "collator_registry", arg0)
	return *ret, err
}

// Collator_registry is a free data retrieval call binding the contract method 0xcc05589c.
//
// Solidity: function collator_registry( address) constant returns(deregistered uint256, pool_index uint256, deposited bool)
func (_SMC *SMCSession) Collator_registry(arg0 common.Address) (struct {
	Deregistered *big.Int
	Pool_index   *big.Int
	Deposited    bool
}, error) {
	return _SMC.Contract.Collator_registry(&_SMC.CallOpts, arg0)
}

// Collator_registry is a free data retrieval call binding the contract method 0xcc05589c.
//
// Solidity: function collator_registry( address) constant returns(deregistered uint256, pool_index uint256, deposited bool)
func (_SMC *SMCCallerSession) Collator_registry(arg0 common.Address) (struct {
	Deregistered *big.Int
	Pool_index   *big.Int
	Deposited    bool
}, error) {
	return _SMC.Contract.Collator_registry(&_SMC.CallOpts, arg0)
}

// Get_eligible_collator is a free data retrieval call binding the contract method 0xf3ae61b7.
//
// Solidity: function get_eligible_collator(shard_id uint256, period uint256) constant returns(address)
func (_SMC *SMCCaller) Get_eligible_collator(opts *bind.CallOpts, shard_id *big.Int, period *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "get_eligible_collator", shard_id, period)
	return *ret0, err
}

// Get_eligible_collator is a free data retrieval call binding the contract method 0xf3ae61b7.
//
// Solidity: function get_eligible_collator(shard_id uint256, period uint256) constant returns(address)
func (_SMC *SMCSession) Get_eligible_collator(shard_id *big.Int, period *big.Int) (common.Address, error) {
	return _SMC.Contract.Get_eligible_collator(&_SMC.CallOpts, shard_id, period)
}

// Get_eligible_collator is a free data retrieval call binding the contract method 0xf3ae61b7.
//
// Solidity: function get_eligible_collator(shard_id uint256, period uint256) constant returns(address)
func (_SMC *SMCCallerSession) Get_eligible_collator(shard_id *big.Int, period *big.Int) (common.Address, error) {
	return _SMC.Contract.Get_eligible_collator(&_SMC.CallOpts, shard_id, period)
}

// Period_head is a free data retrieval call binding the contract method 0x4462ffae.
//
// Solidity: function period_head( int256) constant returns(int256)
func (_SMC *SMCCaller) Period_head(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "period_head", arg0)
	return *ret0, err
}

// Period_head is a free data retrieval call binding the contract method 0x4462ffae.
//
// Solidity: function period_head( int256) constant returns(int256)
func (_SMC *SMCSession) Period_head(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.Period_head(&_SMC.CallOpts, arg0)
}

// Period_head is a free data retrieval call binding the contract method 0x4462ffae.
//
// Solidity: function period_head( int256) constant returns(int256)
func (_SMC *SMCCallerSession) Period_head(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.Period_head(&_SMC.CallOpts, arg0)
}

// Proposer_pool is a free data retrieval call binding the contract method 0x6d9061c3.
//
// Solidity: function proposer_pool( uint256) constant returns(shardId uint256)
func (_SMC *SMCCaller) Proposer_pool(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "proposer_pool", arg0)
	return *ret0, err
}

// Proposer_pool is a free data retrieval call binding the contract method 0x6d9061c3.
//
// Solidity: function proposer_pool( uint256) constant returns(shardId uint256)
func (_SMC *SMCSession) Proposer_pool(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.Proposer_pool(&_SMC.CallOpts, arg0)
}

// Proposer_pool is a free data retrieval call binding the contract method 0x6d9061c3.
//
// Solidity: function proposer_pool( uint256) constant returns(shardId uint256)
func (_SMC *SMCCallerSession) Proposer_pool(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.Proposer_pool(&_SMC.CallOpts, arg0)
}

// Proposer_registry is a free data retrieval call binding the contract method 0x98bd94df.
//
// Solidity: function proposer_registry( address) constant returns(shardId uint256)
func (_SMC *SMCCaller) Proposer_registry(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "proposer_registry", arg0)
	return *ret0, err
}

// Proposer_registry is a free data retrieval call binding the contract method 0x98bd94df.
//
// Solidity: function proposer_registry( address) constant returns(shardId uint256)
func (_SMC *SMCSession) Proposer_registry(arg0 common.Address) (*big.Int, error) {
	return _SMC.Contract.Proposer_registry(&_SMC.CallOpts, arg0)
}

// Proposer_registry is a free data retrieval call binding the contract method 0x98bd94df.
//
// Solidity: function proposer_registry( address) constant returns(shardId uint256)
func (_SMC *SMCCallerSession) Proposer_registry(arg0 common.Address) (*big.Int, error) {
	return _SMC.Contract.Proposer_registry(&_SMC.CallOpts, arg0)
}

// AddHeader is a paid mutator transaction binding the contract method 0x683eee78.
//
// Solidity: function addHeader(_shardId uint256, period uint256, height bytes32, _parent_hash bytes32, chunk_root bytes32, proposer_address address, proposer_bid uint256, proposer_signature bytes) returns(bool)
func (_SMC *SMCTransactor) AddHeader(opts *bind.TransactOpts, _shardId *big.Int, period *big.Int, height [32]byte, _parent_hash [32]byte, chunk_root [32]byte, proposer_address common.Address, proposer_bid *big.Int, proposer_signature []byte) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "addHeader", _shardId, period, height, _parent_hash, chunk_root, proposer_address, proposer_bid, proposer_signature)
}

// AddHeader is a paid mutator transaction binding the contract method 0x683eee78.
//
// Solidity: function addHeader(_shardId uint256, period uint256, height bytes32, _parent_hash bytes32, chunk_root bytes32, proposer_address address, proposer_bid uint256, proposer_signature bytes) returns(bool)
func (_SMC *SMCSession) AddHeader(_shardId *big.Int, period *big.Int, height [32]byte, _parent_hash [32]byte, chunk_root [32]byte, proposer_address common.Address, proposer_bid *big.Int, proposer_signature []byte) (*types.Transaction, error) {
	return _SMC.Contract.AddHeader(&_SMC.TransactOpts, _shardId, period, height, _parent_hash, chunk_root, proposer_address, proposer_bid, proposer_signature)
}

// AddHeader is a paid mutator transaction binding the contract method 0x683eee78.
//
// Solidity: function addHeader(_shardId uint256, period uint256, height bytes32, _parent_hash bytes32, chunk_root bytes32, proposer_address address, proposer_bid uint256, proposer_signature bytes) returns(bool)
func (_SMC *SMCTransactorSession) AddHeader(_shardId *big.Int, period *big.Int, height [32]byte, _parent_hash [32]byte, chunk_root [32]byte, proposer_address common.Address, proposer_bid *big.Int, proposer_signature []byte) (*types.Transaction, error) {
	return _SMC.Contract.AddHeader(&_SMC.TransactOpts, _shardId, period, height, _parent_hash, chunk_root, proposer_address, proposer_bid, proposer_signature)
}

// Compute_header_hash is a paid mutator transaction binding the contract method 0x6e6483f4.
//
// Solidity: function compute_header_hash(shard_id uint256, parent_hash bytes32, chunk_root bytes32, period uint256, proposer_address address, proposer_bid uint256) returns(bytes32)
func (_SMC *SMCTransactor) Compute_header_hash(opts *bind.TransactOpts, shard_id *big.Int, parent_hash [32]byte, chunk_root [32]byte, period *big.Int, proposer_address common.Address, proposer_bid *big.Int) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "compute_header_hash", shard_id, parent_hash, chunk_root, period, proposer_address, proposer_bid)
}

// Compute_header_hash is a paid mutator transaction binding the contract method 0x6e6483f4.
//
// Solidity: function compute_header_hash(shard_id uint256, parent_hash bytes32, chunk_root bytes32, period uint256, proposer_address address, proposer_bid uint256) returns(bytes32)
func (_SMC *SMCSession) Compute_header_hash(shard_id *big.Int, parent_hash [32]byte, chunk_root [32]byte, period *big.Int, proposer_address common.Address, proposer_bid *big.Int) (*types.Transaction, error) {
	return _SMC.Contract.Compute_header_hash(&_SMC.TransactOpts, shard_id, parent_hash, chunk_root, period, proposer_address, proposer_bid)
}

// Compute_header_hash is a paid mutator transaction binding the contract method 0x6e6483f4.
//
// Solidity: function compute_header_hash(shard_id uint256, parent_hash bytes32, chunk_root bytes32, period uint256, proposer_address address, proposer_bid uint256) returns(bytes32)
func (_SMC *SMCTransactorSession) Compute_header_hash(shard_id *big.Int, parent_hash [32]byte, chunk_root [32]byte, period *big.Int, proposer_address common.Address, proposer_bid *big.Int) (*types.Transaction, error) {
	return _SMC.Contract.Compute_header_hash(&_SMC.TransactOpts, shard_id, parent_hash, chunk_root, period, proposer_address, proposer_bid)
}

// Deregister_collator is a paid mutator transaction binding the contract method 0xfae39b4f.
//
// Solidity: function deregister_collator() returns()
func (_SMC *SMCTransactor) Deregister_collator(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "deregister_collator")
}

// Deregister_collator is a paid mutator transaction binding the contract method 0xfae39b4f.
//
// Solidity: function deregister_collator() returns()
func (_SMC *SMCSession) Deregister_collator() (*types.Transaction, error) {
	return _SMC.Contract.Deregister_collator(&_SMC.TransactOpts)
}

// Deregister_collator is a paid mutator transaction binding the contract method 0xfae39b4f.
//
// Solidity: function deregister_collator() returns()
func (_SMC *SMCTransactorSession) Deregister_collator() (*types.Transaction, error) {
	return _SMC.Contract.Deregister_collator(&_SMC.TransactOpts)
}

// Deregister_proposer is a paid mutator transaction binding the contract method 0xff683139.
//
// Solidity: function deregister_proposer() returns()
func (_SMC *SMCTransactor) Deregister_proposer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "deregister_proposer")
}

// Deregister_proposer is a paid mutator transaction binding the contract method 0xff683139.
//
// Solidity: function deregister_proposer() returns()
func (_SMC *SMCSession) Deregister_proposer() (*types.Transaction, error) {
	return _SMC.Contract.Deregister_proposer(&_SMC.TransactOpts)
}

// Deregister_proposer is a paid mutator transaction binding the contract method 0xff683139.
//
// Solidity: function deregister_proposer() returns()
func (_SMC *SMCTransactorSession) Deregister_proposer() (*types.Transaction, error) {
	return _SMC.Contract.Deregister_proposer(&_SMC.TransactOpts)
}

// Proposer_add_balance is a paid mutator transaction binding the contract method 0x5f7d96b5.
//
// Solidity: function proposer_add_balance(shard_id uint256) returns()
func (_SMC *SMCTransactor) Proposer_add_balance(opts *bind.TransactOpts, shard_id *big.Int) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "proposer_add_balance", shard_id)
}

// Proposer_add_balance is a paid mutator transaction binding the contract method 0x5f7d96b5.
//
// Solidity: function proposer_add_balance(shard_id uint256) returns()
func (_SMC *SMCSession) Proposer_add_balance(shard_id *big.Int) (*types.Transaction, error) {
	return _SMC.Contract.Proposer_add_balance(&_SMC.TransactOpts, shard_id)
}

// Proposer_add_balance is a paid mutator transaction binding the contract method 0x5f7d96b5.
//
// Solidity: function proposer_add_balance(shard_id uint256) returns()
func (_SMC *SMCTransactorSession) Proposer_add_balance(shard_id *big.Int) (*types.Transaction, error) {
	return _SMC.Contract.Proposer_add_balance(&_SMC.TransactOpts, shard_id)
}

// Proposer_withdraw_balance is a paid mutator transaction binding the contract method 0x9c56a7e0.
//
// Solidity: function proposer_withdraw_balance(shard_id uint256) returns()
func (_SMC *SMCTransactor) Proposer_withdraw_balance(opts *bind.TransactOpts, shard_id *big.Int) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "proposer_withdraw_balance", shard_id)
}

// Proposer_withdraw_balance is a paid mutator transaction binding the contract method 0x9c56a7e0.
//
// Solidity: function proposer_withdraw_balance(shard_id uint256) returns()
func (_SMC *SMCSession) Proposer_withdraw_balance(shard_id *big.Int) (*types.Transaction, error) {
	return _SMC.Contract.Proposer_withdraw_balance(&_SMC.TransactOpts, shard_id)
}

// Proposer_withdraw_balance is a paid mutator transaction binding the contract method 0x9c56a7e0.
//
// Solidity: function proposer_withdraw_balance(shard_id uint256) returns()
func (_SMC *SMCTransactorSession) Proposer_withdraw_balance(shard_id *big.Int) (*types.Transaction, error) {
	return _SMC.Contract.Proposer_withdraw_balance(&_SMC.TransactOpts, shard_id)
}

// Register_collator is a paid mutator transaction binding the contract method 0xd64e1650.
//
// Solidity: function register_collator() returns(bool)
func (_SMC *SMCTransactor) Register_collator(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "register_collator")
}

// Register_collator is a paid mutator transaction binding the contract method 0xd64e1650.
//
// Solidity: function register_collator() returns(bool)
func (_SMC *SMCSession) Register_collator() (*types.Transaction, error) {
	return _SMC.Contract.Register_collator(&_SMC.TransactOpts)
}

// Register_collator is a paid mutator transaction binding the contract method 0xd64e1650.
//
// Solidity: function register_collator() returns(bool)
func (_SMC *SMCTransactorSession) Register_collator() (*types.Transaction, error) {
	return _SMC.Contract.Register_collator(&_SMC.TransactOpts)
}

// Register_proposer is a paid mutator transaction binding the contract method 0x531b6fc7.
//
// Solidity: function register_proposer() returns(int256)
func (_SMC *SMCTransactor) Register_proposer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "register_proposer")
}

// Register_proposer is a paid mutator transaction binding the contract method 0x531b6fc7.
//
// Solidity: function register_proposer() returns(int256)
func (_SMC *SMCSession) Register_proposer() (*types.Transaction, error) {
	return _SMC.Contract.Register_proposer(&_SMC.TransactOpts)
}

// Register_proposer is a paid mutator transaction binding the contract method 0x531b6fc7.
//
// Solidity: function register_proposer() returns(int256)
func (_SMC *SMCTransactorSession) Register_proposer() (*types.Transaction, error) {
	return _SMC.Contract.Register_proposer(&_SMC.TransactOpts)
}

// Release_collator is a paid mutator transaction binding the contract method 0xd763dfd0.
//
// Solidity: function release_collator() returns()
func (_SMC *SMCTransactor) Release_collator(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "release_collator")
}

// Release_collator is a paid mutator transaction binding the contract method 0xd763dfd0.
//
// Solidity: function release_collator() returns()
func (_SMC *SMCSession) Release_collator() (*types.Transaction, error) {
	return _SMC.Contract.Release_collator(&_SMC.TransactOpts)
}

// Release_collator is a paid mutator transaction binding the contract method 0xd763dfd0.
//
// Solidity: function release_collator() returns()
func (_SMC *SMCTransactorSession) Release_collator() (*types.Transaction, error) {
	return _SMC.Contract.Release_collator(&_SMC.TransactOpts)
}

// Release_proposer is a paid mutator transaction binding the contract method 0x5950d0ca.
//
// Solidity: function release_proposer() returns()
func (_SMC *SMCTransactor) Release_proposer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "release_proposer")
}

// Release_proposer is a paid mutator transaction binding the contract method 0x5950d0ca.
//
// Solidity: function release_proposer() returns()
func (_SMC *SMCSession) Release_proposer() (*types.Transaction, error) {
	return _SMC.Contract.Release_proposer(&_SMC.TransactOpts)
}

// Release_proposer is a paid mutator transaction binding the contract method 0x5950d0ca.
//
// Solidity: function release_proposer() returns()
func (_SMC *SMCTransactorSession) Release_proposer() (*types.Transaction, error) {
	return _SMC.Contract.Release_proposer(&_SMC.TransactOpts)
}

// SMCCollatorDeregisteredIterator is returned from FilterCollatorDeregistered and is used to iterate over the raw logs and unpacked data for CollatorDeregistered events raised by the SMC contract.
type SMCCollatorDeregisteredIterator struct {
	Event *SMCCollatorDeregistered // Event containing the contract specifics and raw log

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
func (it *SMCCollatorDeregisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCCollatorDeregistered)
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
		it.Event = new(SMCCollatorDeregistered)
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
func (it *SMCCollatorDeregisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCCollatorDeregisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCCollatorDeregistered represents a CollatorDeregistered event raised by the SMC contract.
type SMCCollatorDeregistered struct {
	Pool_index *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCollatorDeregistered is a free log retrieval operation binding the contract event 0x9b2d06607fbfb01da25cdaa6312dfbc880589de849cdb80dae0a0e8c87e097c6.
//
// Solidity: event CollatorDeregistered(pool_index uint256)
func (_SMC *SMCFilterer) FilterCollatorDeregistered(opts *bind.FilterOpts) (*SMCCollatorDeregisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "CollatorDeregistered")
	if err != nil {
		return nil, err
	}
	return &SMCCollatorDeregisteredIterator{contract: _SMC.contract, event: "CollatorDeregistered", logs: logs, sub: sub}, nil
}

// WatchCollatorDeregistered is a free log subscription operation binding the contract event 0x9b2d06607fbfb01da25cdaa6312dfbc880589de849cdb80dae0a0e8c87e097c6.
//
// Solidity: event CollatorDeregistered(pool_index uint256)
func (_SMC *SMCFilterer) WatchCollatorDeregistered(opts *bind.WatchOpts, sink chan<- *SMCCollatorDeregistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "CollatorDeregistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCCollatorDeregistered)
				if err := _SMC.contract.UnpackLog(event, "CollatorDeregistered", log); err != nil {
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

// SMCCollatorRegisteredIterator is returned from FilterCollatorRegistered and is used to iterate over the raw logs and unpacked data for CollatorRegistered events raised by the SMC contract.
type SMCCollatorRegisteredIterator struct {
	Event *SMCCollatorRegistered // Event containing the contract specifics and raw log

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
func (it *SMCCollatorRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCCollatorRegistered)
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
		it.Event = new(SMCCollatorRegistered)
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
func (it *SMCCollatorRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCCollatorRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCCollatorRegistered represents a CollatorRegistered event raised by the SMC contract.
type SMCCollatorRegistered struct {
	Collator   common.Address
	Pool_index *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCollatorRegistered is a free log retrieval operation binding the contract event 0xb3cf9a630836c5786f2ca8e1c1db9b946b76f3f7f18aa30514214a9ad16ecfc3.
//
// Solidity: event CollatorRegistered(collator address, pool_index uint256)
func (_SMC *SMCFilterer) FilterCollatorRegistered(opts *bind.FilterOpts) (*SMCCollatorRegisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "CollatorRegistered")
	if err != nil {
		return nil, err
	}
	return &SMCCollatorRegisteredIterator{contract: _SMC.contract, event: "CollatorRegistered", logs: logs, sub: sub}, nil
}

// WatchCollatorRegistered is a free log subscription operation binding the contract event 0xb3cf9a630836c5786f2ca8e1c1db9b946b76f3f7f18aa30514214a9ad16ecfc3.
//
// Solidity: event CollatorRegistered(collator address, pool_index uint256)
func (_SMC *SMCFilterer) WatchCollatorRegistered(opts *bind.WatchOpts, sink chan<- *SMCCollatorRegistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "CollatorRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCCollatorRegistered)
				if err := _SMC.contract.UnpackLog(event, "CollatorRegistered", log); err != nil {
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

// SMCCollatorReleasedIterator is returned from FilterCollatorReleased and is used to iterate over the raw logs and unpacked data for CollatorReleased events raised by the SMC contract.
type SMCCollatorReleasedIterator struct {
	Event *SMCCollatorReleased // Event containing the contract specifics and raw log

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
func (it *SMCCollatorReleasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCCollatorReleased)
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
		it.Event = new(SMCCollatorReleased)
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
func (it *SMCCollatorReleasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCCollatorReleasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCCollatorReleased represents a CollatorReleased event raised by the SMC contract.
type SMCCollatorReleased struct {
	Pool_index *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCollatorReleased is a free log retrieval operation binding the contract event 0x4fcc986c2008cc42fd5a116ab198ba0994004b6758ed1b6052884fc8aa5a6a28.
//
// Solidity: event CollatorReleased(pool_index uint256)
func (_SMC *SMCFilterer) FilterCollatorReleased(opts *bind.FilterOpts) (*SMCCollatorReleasedIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "CollatorReleased")
	if err != nil {
		return nil, err
	}
	return &SMCCollatorReleasedIterator{contract: _SMC.contract, event: "CollatorReleased", logs: logs, sub: sub}, nil
}

// WatchCollatorReleased is a free log subscription operation binding the contract event 0x4fcc986c2008cc42fd5a116ab198ba0994004b6758ed1b6052884fc8aa5a6a28.
//
// Solidity: event CollatorReleased(pool_index uint256)
func (_SMC *SMCFilterer) WatchCollatorReleased(opts *bind.WatchOpts, sink chan<- *SMCCollatorReleased) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "CollatorReleased")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCCollatorReleased)
				if err := _SMC.contract.UnpackLog(event, "CollatorReleased", log); err != nil {
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

// SMCHeaderAddedIterator is returned from FilterHeaderAdded and is used to iterate over the raw logs and unpacked data for HeaderAdded events raised by the SMC contract.
type SMCHeaderAddedIterator struct {
	Event *SMCHeaderAdded // Event containing the contract specifics and raw log

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
func (it *SMCHeaderAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCHeaderAdded)
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
		it.Event = new(SMCHeaderAdded)
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
func (it *SMCHeaderAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCHeaderAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCHeaderAdded represents a HeaderAdded event raised by the SMC contract.
type SMCHeaderAdded struct {
	Shard_id           *big.Int
	Parent_hash        [32]byte
	Chunk_root         [32]byte
	Period             *big.Int
	Height             *big.Int
	Proposer_address   common.Address
	Proposer_bid       *big.Int
	Proposer_signature []byte
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterHeaderAdded is a free log retrieval operation binding the contract event 0xc1e504f1768bb2e62803ca894ced8e4d82ddbd291c4f59c8213e7a36b8da56a9.
//
// Solidity: event HeaderAdded(shard_id indexed uint256, parent_hash bytes32, chunk_root bytes32, period int128, height int128, proposer_address address, proposer_bid uint256, proposer_signature bytes)
func (_SMC *SMCFilterer) FilterHeaderAdded(opts *bind.FilterOpts, shard_id []*big.Int) (*SMCHeaderAddedIterator, error) {

	var shard_idRule []interface{}
	for _, shard_idItem := range shard_id {
		shard_idRule = append(shard_idRule, shard_idItem)
	}

	logs, sub, err := _SMC.contract.FilterLogs(opts, "HeaderAdded", shard_idRule)
	if err != nil {
		return nil, err
	}
	return &SMCHeaderAddedIterator{contract: _SMC.contract, event: "HeaderAdded", logs: logs, sub: sub}, nil
}

// WatchHeaderAdded is a free log subscription operation binding the contract event 0xc1e504f1768bb2e62803ca894ced8e4d82ddbd291c4f59c8213e7a36b8da56a9.
//
// Solidity: event HeaderAdded(shard_id indexed uint256, parent_hash bytes32, chunk_root bytes32, period int128, height int128, proposer_address address, proposer_bid uint256, proposer_signature bytes)
func (_SMC *SMCFilterer) WatchHeaderAdded(opts *bind.WatchOpts, sink chan<- *SMCHeaderAdded, shard_id []*big.Int) (event.Subscription, error) {

	var shard_idRule []interface{}
	for _, shard_idItem := range shard_id {
		shard_idRule = append(shard_idRule, shard_idItem)
	}

	logs, sub, err := _SMC.contract.WatchLogs(opts, "HeaderAdded", shard_idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCHeaderAdded)
				if err := _SMC.contract.UnpackLog(event, "HeaderAdded", log); err != nil {
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

// SMCProposerDeregisteredIterator is returned from FilterProposerDeregistered and is used to iterate over the raw logs and unpacked data for ProposerDeregistered events raised by the SMC contract.
type SMCProposerDeregisteredIterator struct {
	Event *SMCProposerDeregistered // Event containing the contract specifics and raw log

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
func (it *SMCProposerDeregisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCProposerDeregistered)
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
		it.Event = new(SMCProposerDeregistered)
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
func (it *SMCProposerDeregisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCProposerDeregisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCProposerDeregistered represents a ProposerDeregistered event raised by the SMC contract.
type SMCProposerDeregistered struct {
	Index *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterProposerDeregistered is a free log retrieval operation binding the contract event 0xbc0dd14a0f4acad4bf840e15941b010e443a7edaafcc4394fa080a88a362f32b.
//
// Solidity: event ProposerDeregistered(index uint256)
func (_SMC *SMCFilterer) FilterProposerDeregistered(opts *bind.FilterOpts) (*SMCProposerDeregisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "ProposerDeregistered")
	if err != nil {
		return nil, err
	}
	return &SMCProposerDeregisteredIterator{contract: _SMC.contract, event: "ProposerDeregistered", logs: logs, sub: sub}, nil
}

// WatchProposerDeregistered is a free log subscription operation binding the contract event 0xbc0dd14a0f4acad4bf840e15941b010e443a7edaafcc4394fa080a88a362f32b.
//
// Solidity: event ProposerDeregistered(index uint256)
func (_SMC *SMCFilterer) WatchProposerDeregistered(opts *bind.WatchOpts, sink chan<- *SMCProposerDeregistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "ProposerDeregistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCProposerDeregistered)
				if err := _SMC.contract.UnpackLog(event, "ProposerDeregistered", log); err != nil {
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

// SMCProposerRegisteredIterator is returned from FilterProposerRegistered and is used to iterate over the raw logs and unpacked data for ProposerRegistered events raised by the SMC contract.
type SMCProposerRegisteredIterator struct {
	Event *SMCProposerRegistered // Event containing the contract specifics and raw log

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
func (it *SMCProposerRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCProposerRegistered)
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
		it.Event = new(SMCProposerRegistered)
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
func (it *SMCProposerRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCProposerRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCProposerRegistered represents a ProposerRegistered event raised by the SMC contract.
type SMCProposerRegistered struct {
	Pool_index *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterProposerRegistered is a free log retrieval operation binding the contract event 0xb8ab3ee85df29b051303a1468693d1fc8df07a41c749313cb7f7113a321e1327.
//
// Solidity: event ProposerRegistered(pool_index uint256)
func (_SMC *SMCFilterer) FilterProposerRegistered(opts *bind.FilterOpts) (*SMCProposerRegisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "ProposerRegistered")
	if err != nil {
		return nil, err
	}
	return &SMCProposerRegisteredIterator{contract: _SMC.contract, event: "ProposerRegistered", logs: logs, sub: sub}, nil
}

// WatchProposerRegistered is a free log subscription operation binding the contract event 0xb8ab3ee85df29b051303a1468693d1fc8df07a41c749313cb7f7113a321e1327.
//
// Solidity: event ProposerRegistered(pool_index uint256)
func (_SMC *SMCFilterer) WatchProposerRegistered(opts *bind.WatchOpts, sink chan<- *SMCProposerRegistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "ProposerRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCProposerRegistered)
				if err := _SMC.contract.UnpackLog(event, "ProposerRegistered", log); err != nil {
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

// SMCProposerReleasedIterator is returned from FilterProposerReleased and is used to iterate over the raw logs and unpacked data for ProposerReleased events raised by the SMC contract.
type SMCProposerReleasedIterator struct {
	Event *SMCProposerReleased // Event containing the contract specifics and raw log

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
func (it *SMCProposerReleasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCProposerReleased)
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
		it.Event = new(SMCProposerReleased)
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
func (it *SMCProposerReleasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCProposerReleasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCProposerReleased represents a ProposerReleased event raised by the SMC contract.
type SMCProposerReleased struct {
	Index *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterProposerReleased is a free log retrieval operation binding the contract event 0x6597d184b1c015df6d09c20d29722137c8caec0e16ca93c9771f6f39a1a4dfec.
//
// Solidity: event ProposerReleased(index uint256)
func (_SMC *SMCFilterer) FilterProposerReleased(opts *bind.FilterOpts) (*SMCProposerReleasedIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "ProposerReleased")
	if err != nil {
		return nil, err
	}
	return &SMCProposerReleasedIterator{contract: _SMC.contract, event: "ProposerReleased", logs: logs, sub: sub}, nil
}

// WatchProposerReleased is a free log subscription operation binding the contract event 0x6597d184b1c015df6d09c20d29722137c8caec0e16ca93c9771f6f39a1a4dfec.
//
// Solidity: event ProposerReleased(index uint256)
func (_SMC *SMCFilterer) WatchProposerReleased(opts *bind.WatchOpts, sink chan<- *SMCProposerReleased) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "ProposerReleased")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCProposerReleased)
				if err := _SMC.contract.UnpackLog(event, "ProposerReleased", log); err != nil {
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
