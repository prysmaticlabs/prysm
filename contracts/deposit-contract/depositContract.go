// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package depositcontract

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

// DepositContractABI is the input ABI used to generate the binding from.
const DepositContractABI = "[{\"name\":\"Deposit\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"data\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"merkle_tree_index\",\"indexed\":false},{\"type\":\"bytes32[32]\",\"name\":\"branch\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"ChainStart\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"time\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"__init__\",\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"depositThreshold\"},{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"uint256\",\"name\":\"maxDeposit\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"to_bytes8\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"value\"}],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":3474},{\"name\":\"get_deposit_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":30805},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"deposit_input\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":418083},{\"name\":\"chainStarted\",\"outputs\":[{\"type\":\"bool\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":573}]"

// DepositContractBin is the compiled bytecode used for deploying new contracts.
const DepositContractBin = `0x600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a0526060610e036101403934156100a757600080fd5b6101405160025561016051600055610180516001556101a06000601f818352015b60006101a051602081106100db57600080fd5b600360c052602060c02001546020826101c00101526020810190506101a0516020811061010757600080fd5b600360c052602060c02001546020826101c0010152602081019050806101c0526101c0905080516020820120905060605160016101a051018060405190131561014f57600080fd5b809190121561015d57600080fd5b6020811061016a57600080fd5b600360c052602060c020015560605160016101a051018060405190131561019057600080fd5b809190121561019e57600080fd5b602081106101ab57600080fd5b600360c052602060c020015460605160016101a05101806040519013156101d157600080fd5b80919012156101df57600080fd5b602081106101ec57600080fd5b600460c052602060c02001555b81516001018083528114156100c8575b5050610deb56600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a05263b0429c70600051141561017f57602060046101403734156100b457600080fd5b60186008602082066101e001602082840111156100d057600080fd5b60208061020082610140600060046015f15050818152809050905090508051602001806102a0828460006004600a8704601201f161010d57600080fd5b50506102a05160206001820306601f82010390506103006102a0516008818352015b8261030051111561013f5761015b565b6000610300516102c001535b815160010180835281141561012f575b50505060206102805260406102a0510160206001820306601f8201039050610280f3005b63c5f2892f60005114156102b857341561019857600080fd5b6000610140526005546101605261018060006020818352015b600160026101be57600080fd5b60026101605106141561022857600061018051602081106101de57600080fd5b600460c052602060c0200154602082610220010152602081019050610140516020826102200101526020810190508061022052610220905080516020820120905061014052610281565b6000610140516020826101a0010152602081019050610180516020811061024e57600080fd5b600360c052602060c02001546020826101a0010152602081019050806101a0526101a09050805160208201209050610140525b610160600261028f57600080fd5b60028151048152505b81516001018083528114156101b1575b50506101405160005260206000f3005b6398b1e06a6000511415610baf576020600461014037610220600435600401610160376102006004356004013511156102f057600080fd5b633b9aca006103c0526103c05161030657600080fd5b6103c05134046103a0526000546103a051101561032257600080fd5b6001546103a051111561033457600080fd5b6005546103e052426104005260006060610700602463b0429c70610680526103a0516106a05261069c6000305af161036b57600080fd5b61072060088060208461084001018260208501600060046012f150508051820191505060606107e0602463b0429c7061076052610400516107805261077c6000305af16103b757600080fd5b61080060088060208461084001018260208501600060046012f15050805182019150506101606102008060208461084001018260208501600060046045f150508051820191505080610840526108409050805160200180610420828460006004600a8704601201f161042857600080fd5b50506000610aa0526002610ac052610ae060006020818352015b6000610ac05161045157600080fd5b610ac0516103e05160016103e05101101561046b57600080fd5b60016103e051010614151561047f576104eb565b610aa060605160018251018060405190131561049a57600080fd5b80919012156104a857600080fd5b815250610ac0805115156104bd5760006104d7565b60028151600283510204146104d157600080fd5b60028151025b8152505b8151600101808352811415610442575b5050610420805160208201209050610b0052610b2060006020818352015b610aa051610b20511215610570576000610b20516020811061052a57600080fd5b600460c052602060c0200154602082610b40010152602081019050610b0051602082610b4001015260208101905080610b4052610b409050805160208201209050610b00525b5b8151600101808352811415610509575b5050610b0051610aa0516020811061059857600080fd5b600460c052602060c020015560058054600182540110156105b857600080fd5b60018154018155506020610c40600463c5f2892f610be052610bfc6000305af16105e157600080fd5b610c4051610bc0526060610ce0602463b0429c70610c60526103e051610c8052610c7c6000305af161061257600080fd5b610d00805160200180610d40828460006004600a8704601201f161063557600080fd5b5050610bc051610e0052600460c052602060c02054610e60526001600460c052602060c0200154610e80526002600460c052602060c0200154610ea0526003600460c052602060c0200154610ec0526004600460c052602060c0200154610ee0526005600460c052602060c0200154610f00526006600460c052602060c0200154610f20526007600460c052602060c0200154610f40526008600460c052602060c0200154610f60526009600460c052602060c0200154610f8052600a600460c052602060c0200154610fa052600b600460c052602060c0200154610fc052600c600460c052602060c0200154610fe052600d600460c052602060c020015461100052600e600460c052602060c020015461102052600f600460c052602060c0200154611040526010600460c052602060c0200154611060526011600460c052602060c0200154611080526012600460c052602060c02001546110a0526013600460c052602060c02001546110c0526014600460c052602060c02001546110e0526015600460c052602060c0200154611100526016600460c052602060c0200154611120526017600460c052602060c0200154611140526018600460c052602060c0200154611160526019600460c052602060c020015461118052601a600460c052602060c02001546111a052601b600460c052602060c02001546111c052601c600460c052602060c02001546111e052601d600460c052602060c020015461120052601e600460c052602060c020015461122052601f600460c052602060c020015461124052610460610dc052610dc051610e2052610420805160200180610dc051610e0001828460006004600a8704601201f16108b357600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da08151610220818352015b83610da0511015156108f25761090f565b6000610da0516020850101535b81516001018083528114156108e1575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc052610dc051610e4052610d40805160200180610dc051610e0001828460006004600a8704601201f161096657600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516020818352015b83610da0511015156109a4576109c1565b6000610da0516020850101535b8151600101808352811415610993575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc0527fce7a77a358682d6c81f71216fb7fb108b03bc8badbf67f5d131ba5363cbefb42610dc051610e00a16001546103a0511415610bad576006805460018254011015610a3257600080fd5b60018154018155506002546006541415610bac5760206112c0600463c5f2892f6112605261127c6000305af1610a6757600080fd5b6112c0516112e0526060611380602463b0429c7061130052610400516113205261131c6000305af1610a9857600080fd5b6113a08051602001806113e0828460006004600a8704601201f1610abb57600080fd5b50506112e0516114a052604061146052611460516114c0526113e0805160200180611460516114a001828460006004600a8704601201f1610afb57600080fd5b5050611460516114a0015160206001820306601f8201039050611460516114a00161144081516020818352015b8361144051101515610b3957610b56565b6000611440516020850101535b8151600101808352811415610b28575b505050506020611460516114a0015160206001820306601f8201039050611460510101611460527fd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc611460516114a0a160016007555b5b005b63845980e86000511415610bd5573415610bc857600080fd5b60075460005260206000f3005b60006000fd5b610210610deb03610210600039610210610deb036000f3`

// DeployDepositContract deploys a new Ethereum contract, binding an instance of DepositContract to it.
func DeployDepositContract(auth *bind.TransactOpts, backend bind.ContractBackend, depositThreshold *big.Int, minDeposit *big.Int, maxDeposit *big.Int) (common.Address, *types.Transaction, *DepositContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DepositContractBin), backend, depositThreshold, minDeposit, maxDeposit)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DepositContract{DepositContractCaller: DepositContractCaller{contract: contract}, DepositContractTransactor: DepositContractTransactor{contract: contract}, DepositContractFilterer: DepositContractFilterer{contract: contract}}, nil
}

// DepositContract is an auto generated Go binding around an Ethereum contract.
type DepositContract struct {
	DepositContractCaller     // Read-only binding to the contract
	DepositContractTransactor // Write-only binding to the contract
	DepositContractFilterer   // Log filterer for contract events
}

// DepositContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type DepositContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DepositContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DepositContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DepositContractSession struct {
	Contract     *DepositContract  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DepositContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DepositContractCallerSession struct {
	Contract *DepositContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// DepositContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DepositContractTransactorSession struct {
	Contract     *DepositContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// DepositContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type DepositContractRaw struct {
	Contract *DepositContract // Generic contract binding to access the raw methods on
}

// DepositContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DepositContractCallerRaw struct {
	Contract *DepositContractCaller // Generic read-only contract binding to access the raw methods on
}

// DepositContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DepositContractTransactorRaw struct {
	Contract *DepositContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDepositContract creates a new instance of DepositContract, bound to a specific deployed contract.
func NewDepositContract(address common.Address, backend bind.ContractBackend) (*DepositContract, error) {
	contract, err := bindDepositContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DepositContract{DepositContractCaller: DepositContractCaller{contract: contract}, DepositContractTransactor: DepositContractTransactor{contract: contract}, DepositContractFilterer: DepositContractFilterer{contract: contract}}, nil
}

// NewDepositContractCaller creates a new read-only instance of DepositContract, bound to a specific deployed contract.
func NewDepositContractCaller(address common.Address, caller bind.ContractCaller) (*DepositContractCaller, error) {
	contract, err := bindDepositContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DepositContractCaller{contract: contract}, nil
}

// NewDepositContractTransactor creates a new write-only instance of DepositContract, bound to a specific deployed contract.
func NewDepositContractTransactor(address common.Address, transactor bind.ContractTransactor) (*DepositContractTransactor, error) {
	contract, err := bindDepositContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DepositContractTransactor{contract: contract}, nil
}

// NewDepositContractFilterer creates a new log filterer instance of DepositContract, bound to a specific deployed contract.
func NewDepositContractFilterer(address common.Address, filterer bind.ContractFilterer) (*DepositContractFilterer, error) {
	contract, err := bindDepositContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DepositContractFilterer{contract: contract}, nil
}

// bindDepositContract binds a generic wrapper to an already deployed contract.
func bindDepositContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DepositContract *DepositContractRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _DepositContract.Contract.DepositContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DepositContract *DepositContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DepositContract.Contract.DepositContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DepositContract *DepositContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DepositContract.Contract.DepositContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DepositContract *DepositContractCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _DepositContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DepositContract *DepositContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DepositContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DepositContract *DepositContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DepositContract.Contract.contract.Transact(opts, method, params...)
}

// ChainStarted is a free data retrieval call binding the contract method 0x845980e8.
//
// Solidity: function chainStarted() constant returns(out bool)
func (_DepositContract *DepositContractCaller) ChainStarted(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "chainStarted")
	return *ret0, err
}

// ChainStarted is a free data retrieval call binding the contract method 0x845980e8.
//
// Solidity: function chainStarted() constant returns(out bool)
func (_DepositContract *DepositContractSession) ChainStarted() (bool, error) {
	return _DepositContract.Contract.ChainStarted(&_DepositContract.CallOpts)
}

// ChainStarted is a free data retrieval call binding the contract method 0x845980e8.
//
// Solidity: function chainStarted() constant returns(out bool)
func (_DepositContract *DepositContractCallerSession) ChainStarted() (bool, error) {
	return _DepositContract.Contract.ChainStarted(&_DepositContract.CallOpts)
}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() constant returns(out bytes32)
func (_DepositContract *DepositContractCaller) GetDepositRoot(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "get_deposit_root")
	return *ret0, err
}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() constant returns(out bytes32)
func (_DepositContract *DepositContractSession) GetDepositRoot() ([32]byte, error) {
	return _DepositContract.Contract.GetDepositRoot(&_DepositContract.CallOpts)
}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() constant returns(out bytes32)
func (_DepositContract *DepositContractCallerSession) GetDepositRoot() ([32]byte, error) {
	return _DepositContract.Contract.GetDepositRoot(&_DepositContract.CallOpts)
}

// ToBytes8 is a free data retrieval call binding the contract method 0xb0429c70.
//
// Solidity: function to_bytes8(value uint256) constant returns(out bytes)
func (_DepositContract *DepositContractCaller) ToBytes8(opts *bind.CallOpts, value *big.Int) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "to_bytes8", value)
	return *ret0, err
}

// ToBytes8 is a free data retrieval call binding the contract method 0xb0429c70.
//
// Solidity: function to_bytes8(value uint256) constant returns(out bytes)
func (_DepositContract *DepositContractSession) ToBytes8(value *big.Int) ([]byte, error) {
	return _DepositContract.Contract.ToBytes8(&_DepositContract.CallOpts, value)
}

// ToBytes8 is a free data retrieval call binding the contract method 0xb0429c70.
//
// Solidity: function to_bytes8(value uint256) constant returns(out bytes)
func (_DepositContract *DepositContractCallerSession) ToBytes8(value *big.Int) ([]byte, error) {
	return _DepositContract.Contract.ToBytes8(&_DepositContract.CallOpts, value)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(deposit_input bytes) returns()
func (_DepositContract *DepositContractTransactor) Deposit(opts *bind.TransactOpts, deposit_input []byte) (*types.Transaction, error) {
	return _DepositContract.contract.Transact(opts, "deposit", deposit_input)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(deposit_input bytes) returns()
func (_DepositContract *DepositContractSession) Deposit(deposit_input []byte) (*types.Transaction, error) {
	return _DepositContract.Contract.Deposit(&_DepositContract.TransactOpts, deposit_input)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(deposit_input bytes) returns()
func (_DepositContract *DepositContractTransactorSession) Deposit(deposit_input []byte) (*types.Transaction, error) {
	return _DepositContract.Contract.Deposit(&_DepositContract.TransactOpts, deposit_input)
}

// DepositContractChainStartIterator is returned from FilterChainStart and is used to iterate over the raw logs and unpacked data for ChainStart events raised by the DepositContract contract.
type DepositContractChainStartIterator struct {
	Event *DepositContractChainStart // Event containing the contract specifics and raw log

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
func (it *DepositContractChainStartIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DepositContractChainStart)
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
		it.Event = new(DepositContractChainStart)
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
func (it *DepositContractChainStartIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DepositContractChainStartIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DepositContractChainStart represents a ChainStart event raised by the DepositContract contract.
type DepositContractChainStart struct {
	DepositRoot [32]byte
	Time        []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChainStart is a free log retrieval operation binding the contract event 0xd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc.
//
// Solidity: e ChainStart(deposit_root bytes32, time bytes)
func (_DepositContract *DepositContractFilterer) FilterChainStart(opts *bind.FilterOpts) (*DepositContractChainStartIterator, error) {

	logs, sub, err := _DepositContract.contract.FilterLogs(opts, "ChainStart")
	if err != nil {
		return nil, err
	}
	return &DepositContractChainStartIterator{contract: _DepositContract.contract, event: "ChainStart", logs: logs, sub: sub}, nil
}

// WatchChainStart is a free log subscription operation binding the contract event 0xd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc.
//
// Solidity: e ChainStart(deposit_root bytes32, time bytes)
func (_DepositContract *DepositContractFilterer) WatchChainStart(opts *bind.WatchOpts, sink chan<- *DepositContractChainStart) (event.Subscription, error) {

	logs, sub, err := _DepositContract.contract.WatchLogs(opts, "ChainStart")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DepositContractChainStart)
				if err := _DepositContract.contract.UnpackLog(event, "ChainStart", log); err != nil {
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

// DepositContractDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the DepositContract contract.
type DepositContractDepositIterator struct {
	Event *DepositContractDeposit // Event containing the contract specifics and raw log

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
func (it *DepositContractDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DepositContractDeposit)
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
		it.Event = new(DepositContractDeposit)
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
func (it *DepositContractDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DepositContractDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DepositContractDeposit represents a Deposit event raised by the DepositContract contract.
type DepositContractDeposit struct {
	DepositRoot     [32]byte
	Data            []byte
	MerkleTreeIndex []byte
	Branch          [32][32]byte
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xce7a77a358682d6c81f71216fb7fb108b03bc8badbf67f5d131ba5363cbefb42.
//
// Solidity: e Deposit(deposit_root bytes32, data bytes, merkle_tree_index bytes, branch bytes32[32])
func (_DepositContract *DepositContractFilterer) FilterDeposit(opts *bind.FilterOpts) (*DepositContractDepositIterator, error) {

	logs, sub, err := _DepositContract.contract.FilterLogs(opts, "Deposit")
	if err != nil {
		return nil, err
	}
	return &DepositContractDepositIterator{contract: _DepositContract.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xce7a77a358682d6c81f71216fb7fb108b03bc8badbf67f5d131ba5363cbefb42.
//
// Solidity: e Deposit(deposit_root bytes32, data bytes, merkle_tree_index bytes, branch bytes32[32])
func (_DepositContract *DepositContractFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *DepositContractDeposit) (event.Subscription, error) {

	logs, sub, err := _DepositContract.contract.WatchLogs(opts, "Deposit")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DepositContractDeposit)
				if err := _DepositContract.contract.UnpackLog(event, "Deposit", log); err != nil {
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
