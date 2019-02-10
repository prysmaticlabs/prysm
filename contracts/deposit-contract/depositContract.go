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

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// DepositContractABI is the input ABI used to generate the binding from.
const DepositContractABI = "[{\"name\":\"Deposit\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"data\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"merkle_tree_index\",\"indexed\":false},{\"type\":\"bytes32[32]\",\"name\":\"branch\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"ChainStart\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"time\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"__init__\",\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"depositThreshold\"},{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"uint256\",\"name\":\"maxDeposit\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"to_little_endian_64\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"value\"}],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":9588},{\"name\":\"get_deposit_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":30805},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"deposit_input\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":411346},{\"name\":\"chainStarted\",\"outputs\":[{\"type\":\"bool\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":573}]"

// DepositContractBin is the compiled bytecode used for deploying new contracts.
const DepositContractBin = `0x600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a05260606110ca6101403934156100a757600080fd5b6101405160025561016051600055610180516001556101a06000601f818352015b60006101a051602081106100db57600080fd5b600360c052602060c02001546020826101c00101526020810190506101a0516020811061010757600080fd5b600360c052602060c02001546020826101c0010152602081019050806101c0526101c0905080516020820120905060605160016101a051018060405190131561014f57600080fd5b809190121561015d57600080fd5b6020811061016a57600080fd5b600360c052602060c020015560605160016101a051018060405190131561019057600080fd5b809190121561019e57600080fd5b602081106101ab57600080fd5b600360c052602060c020015460605160016101a05101806040519013156101d157600080fd5b80919012156101df57600080fd5b602081106101ec57600080fd5b600460c052602060c02001555b81516001018083528114156100c8575b50506110b256600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a0526380673289600051141561037157602060046101403734156100b457600080fd5b67ffffffffffffffff6101405111156100cc57600080fd5b6018600860208206610260016000610140516020826102000101526020810190508061020052610200905051828401111561010657600080fd5b602080610280826020602088068803016000610140516020826102000101526020810190508061020052610200905001600060046015f1505081815280905090509050805160200180610160828460006004600a8704601201f161016957600080fd5b505061016080602001516000825180602090131561018657600080fd5b809190121561019457600080fd5b806020036101000a82049050905090506102e05260006103005261032060006008818352015b61030051600860008112156101d7578060000360020a82046101de565b8060020a82025b905090506103005260ff6102e051166103405261030051610340516103005101101561020957600080fd5b610340516103005101610300526102e0517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff86000811215610252578060000360020a8204610259565b8060020a82025b905090506102e0525b81516001018083528114156101ba575b50506018600860208206610400016000610300516020826103a0010152602081019050806103a0526103a090505182840111156102ae57600080fd5b602080610420826020602088068803016000610300516020826103a0010152602081019050806103a0526103a0905001600060046015f15050818152809050905090508051602001806104c0828460006004600a8704601201f161031157600080fd5b50506105206104c0516008818352015b60086105205111156103325761034e565b6000610520516104e001535b8151600101808352811415610321575b505060206104a05260406104c0510160206001820306601f82010390506104a0f3005b63c5f2892f60005114156104aa57341561038a57600080fd5b6000610140526005546101605261018060006020818352015b600160026103b057600080fd5b60026101605106141561041a57600061018051602081106103d057600080fd5b600460c052602060c0200154602082610220010152602081019050610140516020826102200101526020810190508061022052610220905080516020820120905061014052610473565b6000610140516020826101a0010152602081019050610180516020811061044057600080fd5b600360c052602060c02001546020826101a0010152602081019050806101a0526101a09050805160208201209050610140525b610160600261048157600080fd5b60028151048152505b81516001018083528114156103a3575b50506101405160005260206000f3005b6398b1e06a6000511415610e76576020600461014037610220600435600401610160376102006004356004013511156104e257600080fd5b633b9aca00600054023410156104f757600080fd5b633b9aca006001540234111561050c57600080fd5b6005546103a05260606104a06024638067328961042052633b9aca0061053157600080fd5b633b9aca0034046104405261043c6000305af161054d57600080fd5b6104c08051602001806103c0828460006004600a8704601201f161057057600080fd5b505060606105e06024638067328961056052426105805261057c6000305af161059857600080fd5b610600805160200180610500828460006004600a8704601201f16105bb57600080fd5b505060006103c06008806020846108a001018260208501600060046012f15050805182019150506105006008806020846108a001018260208501600060046012f1505080518201915050610160610200806020846108a001018260208501600060046045f1505080518201915050806108a0526108a09050805160200180610640828460006004600a8704601201f161065357600080fd5b50506060610be060246380673289610b60526103a051610b8052610b7c6000305af161067e57600080fd5b610c00805160200180610b00828460006004600a8704601201f16106a157600080fd5b50506000610c40526002610c6052610c8060006020818352015b6000610c60516106ca57600080fd5b610c60516103a05160016103a0510110156106e457600080fd5b60016103a05101061415156106f857610764565b610c4060605160018251018060405190131561071357600080fd5b809190121561072157600080fd5b815250610c6080511515610736576000610750565b600281516002835102041461074a57600080fd5b60028151025b8152505b81516001018083528114156106bb575b5050610640805160208201209050610ca052610cc060006020818352015b610c4051610cc05112156107e9576000610cc051602081106107a357600080fd5b600460c052602060c0200154602082610ce0010152602081019050610ca051602082610ce001015260208101905080610ce052610ce09050805160208201209050610ca0525b5b8151600101808352811415610782575b5050610ca051610c40516020811061081157600080fd5b600460c052602060c0200155600580546001825401101561083157600080fd5b60018154018155506020610de0600463c5f2892f610d8052610d9c6000305af161085a57600080fd5b610de051610d6052610d6051610e6052600460c052602060c02054610ec0526001600460c052602060c0200154610ee0526002600460c052602060c0200154610f00526003600460c052602060c0200154610f20526004600460c052602060c0200154610f40526005600460c052602060c0200154610f60526006600460c052602060c0200154610f80526007600460c052602060c0200154610fa0526008600460c052602060c0200154610fc0526009600460c052602060c0200154610fe052600a600460c052602060c020015461100052600b600460c052602060c020015461102052600c600460c052602060c020015461104052600d600460c052602060c020015461106052600e600460c052602060c020015461108052600f600460c052602060c02001546110a0526010600460c052602060c02001546110c0526011600460c052602060c02001546110e0526012600460c052602060c0200154611100526013600460c052602060c0200154611120526014600460c052602060c0200154611140526015600460c052602060c0200154611160526016600460c052602060c0200154611180526017600460c052602060c02001546111a0526018600460c052602060c02001546111c0526019600460c052602060c02001546111e052601a600460c052602060c020015461120052601b600460c052602060c020015461122052601c600460c052602060c020015461124052601d600460c052602060c020015461126052601e600460c052602060c020015461128052601f600460c052602060c02001546112a052610460610e2052610e2051610e8052610640805160200180610e2051610e6001828460006004600a8704601201f1610ade57600080fd5b5050610e2051610e6001610e008151610220818352015b610220610e0051101515610b0857610b25565b6000610e00516020850101535b8151600101808352811415610af5575b5050506020610e2051610e60015160206001820306601f8201039050610e20510101610e2052610e2051610ea052610b00805160200180610e2051610e6001828460006004600a8704601201f1610b7b57600080fd5b5050610e2051610e6001610e0081516020818352015b6020610e0051101515610ba357610bc0565b6000610e00516020850101535b8151600101808352811415610b91575b5050506020610e2051610e60015160206001820306601f8201039050610e20510101610e20527fce7a77a358682d6c81f71216fb7fb108b03bc8badbf67f5d131ba5363cbefb42610e2051610e60a1633b9aca0060015402341415610e74576006805460018254011015610c3357600080fd5b60018154018155506002546006541415610e7357426112e052426113005262015180610c5e57600080fd5b6201518061130051066112e0511015610c7657600080fd5b426113005262015180610c8857600080fd5b6201518061130051066112e0510362015180426112e052426113005262015180610cb157600080fd5b6201518061130051066112e0511015610cc957600080fd5b426113005262015180610cdb57600080fd5b6201518061130051066112e05103011015610cf557600080fd5b62015180426112e052426113005262015180610d1057600080fd5b6201518061130051066112e0511015610d2857600080fd5b426113005262015180610d3a57600080fd5b6201518061130051066112e05103016112c052606061140060246380673289611380526112c0516113a05261139c6000305af1610d7657600080fd5b611420805160200180611320828460006004600a8704601201f1610d9957600080fd5b5050610d60516114c052604061148052611480516114e052611320805160200180611480516114c001828460006004600a8704601201f1610dd957600080fd5b5050611480516114c00161146081516020818352015b602061146051101515610e0157610e1e565b6000611460516020850101535b8151600101808352811415610def575b5050506020611480516114c0015160206001820306601f8201039050611480510101611480527fd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc611480516114c0a160016007555b5b005b63845980e86000511415610e9c573415610e8f57600080fd5b60075460005260206000f3005b60006000fd5b6102106110b2036102106000396102106110b2036000f3`

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
// Solidity: function chainStarted() constant returns(bool out)
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
// Solidity: function chainStarted() constant returns(bool out)
func (_DepositContract *DepositContractSession) ChainStarted() (bool, error) {
	return _DepositContract.Contract.ChainStarted(&_DepositContract.CallOpts)
}

// ChainStarted is a free data retrieval call binding the contract method 0x845980e8.
//
// Solidity: function chainStarted() constant returns(bool out)
func (_DepositContract *DepositContractCallerSession) ChainStarted() (bool, error) {
	return _DepositContract.Contract.ChainStarted(&_DepositContract.CallOpts)
}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() constant returns(bytes32 out)
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
// Solidity: function get_deposit_root() constant returns(bytes32 out)
func (_DepositContract *DepositContractSession) GetDepositRoot() ([32]byte, error) {
	return _DepositContract.Contract.GetDepositRoot(&_DepositContract.CallOpts)
}

// GetDepositRoot is a free data retrieval call binding the contract method 0xc5f2892f.
//
// Solidity: function get_deposit_root() constant returns(bytes32 out)
func (_DepositContract *DepositContractCallerSession) GetDepositRoot() ([32]byte, error) {
	return _DepositContract.Contract.GetDepositRoot(&_DepositContract.CallOpts)
}

// ToLittleEndian64 is a free data retrieval call binding the contract method 0x80673289.
//
// Solidity: function to_little_endian_64(uint256 value) constant returns(bytes out)
func (_DepositContract *DepositContractCaller) ToLittleEndian64(opts *bind.CallOpts, value *big.Int) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "to_little_endian_64", value)
	return *ret0, err
}

// ToLittleEndian64 is a free data retrieval call binding the contract method 0x80673289.
//
// Solidity: function to_little_endian_64(uint256 value) constant returns(bytes out)
func (_DepositContract *DepositContractSession) ToLittleEndian64(value *big.Int) ([]byte, error) {
	return _DepositContract.Contract.ToLittleEndian64(&_DepositContract.CallOpts, value)
}

// ToLittleEndian64 is a free data retrieval call binding the contract method 0x80673289.
//
// Solidity: function to_little_endian_64(uint256 value) constant returns(bytes out)
func (_DepositContract *DepositContractCallerSession) ToLittleEndian64(value *big.Int) ([]byte, error) {
	return _DepositContract.Contract.ToLittleEndian64(&_DepositContract.CallOpts, value)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(bytes deposit_input) returns()
func (_DepositContract *DepositContractTransactor) Deposit(opts *bind.TransactOpts, deposit_input []byte) (*types.Transaction, error) {
	return _DepositContract.contract.Transact(opts, "deposit", deposit_input)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(bytes deposit_input) returns()
func (_DepositContract *DepositContractSession) Deposit(deposit_input []byte) (*types.Transaction, error) {
	return _DepositContract.Contract.Deposit(&_DepositContract.TransactOpts, deposit_input)
}

// Deposit is a paid mutator transaction binding the contract method 0x98b1e06a.
//
// Solidity: function deposit(bytes deposit_input) returns()
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
// Solidity: event ChainStart(bytes32 deposit_root, bytes time)
func (_DepositContract *DepositContractFilterer) FilterChainStart(opts *bind.FilterOpts) (*DepositContractChainStartIterator, error) {

	logs, sub, err := _DepositContract.contract.FilterLogs(opts, "ChainStart")
	if err != nil {
		return nil, err
	}
	return &DepositContractChainStartIterator{contract: _DepositContract.contract, event: "ChainStart", logs: logs, sub: sub}, nil
}

// WatchChainStart is a free log subscription operation binding the contract event 0xd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc.
//
// Solidity: event ChainStart(bytes32 deposit_root, bytes time)
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
// Solidity: event Deposit(bytes32 deposit_root, bytes data, bytes merkle_tree_index, bytes32[32] branch)
func (_DepositContract *DepositContractFilterer) FilterDeposit(opts *bind.FilterOpts) (*DepositContractDepositIterator, error) {

	logs, sub, err := _DepositContract.contract.FilterLogs(opts, "Deposit")
	if err != nil {
		return nil, err
	}
	return &DepositContractDepositIterator{contract: _DepositContract.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xce7a77a358682d6c81f71216fb7fb108b03bc8badbf67f5d131ba5363cbefb42.
//
// Solidity: event Deposit(bytes32 deposit_root, bytes data, bytes merkle_tree_index, bytes32[32] branch)
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
