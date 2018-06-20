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
const SMCABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"shardCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"currentVote\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"},{\"name\":\"_period\",\"type\":\"uint256\"},{\"name\":\"_index\",\"type\":\"uint256\"},{\"name\":\"_chunkRoot\",\"type\":\"bytes32\"}],\"name\":\"submitVote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"deregisterNotary\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"},{\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"hasVoted\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"}],\"name\":\"getNotaryInCommittee\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"registerNotary\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"notaryRegistry\",\"outputs\":[{\"name\":\"deregisteredPeriod\",\"type\":\"uint256\"},{\"name\":\"poolIndex\",\"type\":\"uint256\"},{\"name\":\"balance\",\"type\":\"uint256\"},{\"name\":\"deposited\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"},{\"name\":\"_period\",\"type\":\"uint256\"},{\"name\":\"_chunkRoot\",\"type\":\"bytes32\"}],\"name\":\"addHeader\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"lastSubmittedCollation\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"lastApprovedCollation\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"releaseNotary\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"notaryPool\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"}],\"name\":\"getVoteCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"CHALLENGE_PERIOD\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"collationRecords\",\"outputs\":[{\"name\":\"chunkRoot\",\"type\":\"bytes32\"},{\"name\":\"proposer\",\"type\":\"address\"},{\"name\":\"isElected\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"notaryPoolLength\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"shardId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"chunkRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"period\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"proposerAddress\",\"type\":\"address\"}],\"name\":\"HeaderAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"notary\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"poolIndex\",\"type\":\"uint256\"}],\"name\":\"NotaryRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"notary\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"poolIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"deregisteredPeriod\",\"type\":\"uint256\"}],\"name\":\"NotaryDeregistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"notary\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"poolIndex\",\"type\":\"uint256\"}],\"name\":\"NotaryReleased\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"shardId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"chunkRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"period\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"notaryAddress\",\"type\":\"address\"}],\"name\":\"VoteSubmitted\",\"type\":\"event\"}]"

// SMCBin is the compiled bytecode used for deploying new contracts.
const SMCBin = `0x60806040526064600c5534801561001557600080fd5b50610cf3806100256000396000f3006080604052600436106100f05763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166304e9c77a81146100f55780630c8da4cc1461011c5780634f33ffa01461013457806358377bd11461015757806364390ff11461016c578063673221af1461019b57806368e9513e146101cf5780636bdd3271146101d757806375bd99121461022057806383ceeabe1461023e57806397d369a2146102565780639910851d1461026e578063a81f451014610283578063b2c2f2e81461029b578063c3a079ed146102b3578063e9e0b683146102c8578063f6f67d361461030b575b600080fd5b34801561010157600080fd5b5061010a610320565b60408051918252519081900360200190f35b34801561012857600080fd5b5061010a600435610326565b34801561014057600080fd5b50610155600435602435604435606435610338565b005b34801561016357600080fd5b506101556104c3565b34801561017857600080fd5b506101876004356024356105e4565b604080519115158252519081900360200190f35b3480156101a757600080fd5b506101b3600435610607565b60408051600160a060020a039092168252519081900360200190f35b6101556106b7565b3480156101e357600080fd5b506101f8600160a060020a0360043516610870565b6040805194855260208501939093528383019190915215156060830152519081900360800190f35b34801561022c57600080fd5b5061015560043560243560443561089b565b34801561024a57600080fd5b5061010a6004356109e9565b34801561026257600080fd5b5061010a6004356109fb565b34801561027a57600080fd5b50610155610a0d565b34801561028f57600080fd5b506101b3600435610b38565b3480156102a757600080fd5b5061010a600435610b60565b3480156102bf57600080fd5b5061010a610b75565b3480156102d457600080fd5b506102e3600435602435610b7a565b60408051938452600160a060020a039092166020840152151582820152519081900360600190f35b34801561031757600080fd5b5061010a610bc4565b600c5481565b60036020526000908152604090205481565b600080851015801561034b5750600c5485105b151561035657600080fd5b60054304841461036557600080fd5b600085815260056020526040902054841461037f57600080fd5b6087831061038c57600080fd5b600085815260046020908152604080832087845290915290205482146103b157600080fd5b3360009081526001602052604090206003015460ff1615156103d257600080fd5b6103dc85846105e4565b156103e657600080fd5b336103f086610607565b600160a060020a03161461040357600080fd5b61040d8584610bca565b61041685610b60565b9050605a811061047a576000858152600660209081526040808320879055600482528083208784529091529020600101805474ff00000000000000000000000000000000000000001916740100000000000000000000000000000000000000001790555b60408051838152602081018690523381830152905186917fc99370212b708f699fb6945a17eb34d0fc1ccd5b45d88f4d9682593a45d6e833919081900360600190a25050505050565b33600081815260016020819052604082209081015460039091015490919060ff1615156104ef57600080fd5b82600160a060020a031660008381548110151561050857fe5b600091825260209091200154600160a060020a03161461052757600080fd5b61052f610bee565b50600160a060020a0382166000908152600160205260409020600543049081905561055982610c11565b600080548390811061056757fe5b600091825260209182902001805473ffffffffffffffffffffffffffffffffffffffff191690556002805460001901905560408051600160a060020a0386168152918201849052818101839052517f90e5afdc8fd31453dcf6e37154fa117ddf3b0324c96c65015563df9d5e4b5a759181900360600190a1505050565b60009182526003602052604090912054600160ff9290920360020a900481161490565b600060054304818080808061061a610bee565b600b5486111561062e57600a549450610634565b60095494505b3360009081526001602081815260409283902090910154825160001960058b020180408083529382018390528185018d905293519081900360600190209096509194509250859081151561068457fe5b06905060008181548110151561069657fe5b600091825260209091200154600160a060020a031698975050505050505050565b3360008181526001602052604081206003015460ff16156106d757600080fd5b34683635c9adc5dea00000146106ec57600080fd5b6106f4610bee565b6106fc610c82565b1561075f5750600254600080546001810182559080527f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56301805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0384161790556107a8565b610767610c89565b90508160008281548110151561077957fe5b9060005260206000200160006101000a815481600160a060020a030219169083600160a060020a031602179055505b600280546001908101825560408051608081018252600080825260208083018781523484860190815260608501878152600160a060020a038b168552928790529490922092518355905193820193909355905192810192909255516003909101805460ff1916911515919091179055600a5481106108285760018101600a555b60408051600160a060020a03841681526020810183905281517fa4fe15c53db34d35a5117acc26c27a2653dc68e2dadfc21ed211e38b7864d7a7929181900390910190a15050565b6001602081905260009182526040909120805491810154600282015460039092015490919060ff1684565b600083101580156108ad5750600c5483105b15156108b857600080fd5b6005430482146108c757600080fd5b60008381526005602052604090205482116108e157600080fd5b6108e9610bee565b604080516060808201835283825233602080840182815260008587018181528a8252600484528782208a835284528782209651875591516001909601805492511515740100000000000000000000000000000000000000000274ff000000000000000000000000000000000000000019600160a060020a0390981673ffffffffffffffffffffffffffffffffffffffff199094169390931796909616919091179094558784526005808252858520439190910490556003815284842093909355835185815292830186905282840152915185927f2d0a86178d2fd307b47be157a766e6bee19bc26161c32f9781ee0e818636f09c928290030190a2505050565b60056020526000908152604090205481565b60066020526000908152604090205481565b33600081815260016020819052604082208082015460039091015490929160ff909116151514610a3c57600080fd5b600160a060020a0383166000908152600160205260409020541515610a6057600080fd5b600160a060020a038316600090815260016020526040902054613f00016005430411610a8b57600080fd5b50600160a060020a0382166000818152600160208190526040808320600281018054858355938201859055849055600301805460ff191690555190929183156108fc02918491818181858888f19350505050158015610aee573d6000803e3d6000fd5b5060408051600160a060020a03851681526020810184905281517faee20171b64b7f3360a142659094ce929970d6963dcea8c34a9bf1ece8033680929181900390910190a1505050565b6000805482908110610b4657fe5b600091825260209091200154600160a060020a0316905081565b60009081526003602052604090205460ff1690565b601981565b600460209081526000928352604080842090915290825290208054600190910154600160a060020a0381169074010000000000000000000000000000000000000000900460ff1683565b60025481565b600091825260036020526040909120805460ff9290920360020a9091176001019055565b600b546005430490811015610c0257610c0e565b600a54600955600b8190555b50565b6008546007541415610c5757600780546001810182556000919091527fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c68801819055610c76565b806007600854815481101515610c6957fe5b6000918252602090912001555b50600880546001019055565b6008541590565b60006001600854111515610c9c57600080fd5b600880546000190190819055600780549091908110610cb757fe5b90600052602060002001549050905600a165627a7a723058207b52440d35fdaa4b38b52691c19ed386efd12e7627da04774cddcc1470d8a1c50029`

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

// CHALLENGEPERIOD is a free data retrieval call binding the contract method 0xc3a079ed.
//
// Solidity: function CHALLENGE_PERIOD() constant returns(uint256)
func (_SMC *SMCCaller) CHALLENGEPERIOD(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "CHALLENGE_PERIOD")
	return *ret0, err
}

// CHALLENGEPERIOD is a free data retrieval call binding the contract method 0xc3a079ed.
//
// Solidity: function CHALLENGE_PERIOD() constant returns(uint256)
func (_SMC *SMCSession) CHALLENGEPERIOD() (*big.Int, error) {
	return _SMC.Contract.CHALLENGEPERIOD(&_SMC.CallOpts)
}

// CHALLENGEPERIOD is a free data retrieval call binding the contract method 0xc3a079ed.
//
// Solidity: function CHALLENGE_PERIOD() constant returns(uint256)
func (_SMC *SMCCallerSession) CHALLENGEPERIOD() (*big.Int, error) {
	return _SMC.Contract.CHALLENGEPERIOD(&_SMC.CallOpts)
}

// CollationRecords is a free data retrieval call binding the contract method 0xe9e0b683.
//
// Solidity: function collationRecords( uint256,  uint256) constant returns(chunkRoot bytes32, proposer address, isElected bool)
func (_SMC *SMCCaller) CollationRecords(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
}, error) {
	ret := new(struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
	})
	out := ret
	err := _SMC.contract.Call(opts, out, "collationRecords", arg0, arg1)
	return *ret, err
}

// CollationRecords is a free data retrieval call binding the contract method 0xe9e0b683.
//
// Solidity: function collationRecords( uint256,  uint256) constant returns(chunkRoot bytes32, proposer address, isElected bool)
func (_SMC *SMCSession) CollationRecords(arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
}, error) {
	return _SMC.Contract.CollationRecords(&_SMC.CallOpts, arg0, arg1)
}

// CollationRecords is a free data retrieval call binding the contract method 0xe9e0b683.
//
// Solidity: function collationRecords( uint256,  uint256) constant returns(chunkRoot bytes32, proposer address, isElected bool)
func (_SMC *SMCCallerSession) CollationRecords(arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
}, error) {
	return _SMC.Contract.CollationRecords(&_SMC.CallOpts, arg0, arg1)
}

// CurrentVote is a free data retrieval call binding the contract method 0x0c8da4cc.
//
// Solidity: function currentVote( uint256) constant returns(bytes32)
func (_SMC *SMCCaller) CurrentVote(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "currentVote", arg0)
	return *ret0, err
}

// CurrentVote is a free data retrieval call binding the contract method 0x0c8da4cc.
//
// Solidity: function currentVote( uint256) constant returns(bytes32)
func (_SMC *SMCSession) CurrentVote(arg0 *big.Int) ([32]byte, error) {
	return _SMC.Contract.CurrentVote(&_SMC.CallOpts, arg0)
}

// CurrentVote is a free data retrieval call binding the contract method 0x0c8da4cc.
//
// Solidity: function currentVote( uint256) constant returns(bytes32)
func (_SMC *SMCCallerSession) CurrentVote(arg0 *big.Int) ([32]byte, error) {
	return _SMC.Contract.CurrentVote(&_SMC.CallOpts, arg0)
}

// GetNotaryInCommittee is a free data retrieval call binding the contract method 0x673221af.
//
// Solidity: function getNotaryInCommittee(_shardId uint256) constant returns(address)
func (_SMC *SMCCaller) GetNotaryInCommittee(opts *bind.CallOpts, _shardId *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "getNotaryInCommittee", _shardId)
	return *ret0, err
}

// GetNotaryInCommittee is a free data retrieval call binding the contract method 0x673221af.
//
// Solidity: function getNotaryInCommittee(_shardId uint256) constant returns(address)
func (_SMC *SMCSession) GetNotaryInCommittee(_shardId *big.Int) (common.Address, error) {
	return _SMC.Contract.GetNotaryInCommittee(&_SMC.CallOpts, _shardId)
}

// GetNotaryInCommittee is a free data retrieval call binding the contract method 0x673221af.
//
// Solidity: function getNotaryInCommittee(_shardId uint256) constant returns(address)
func (_SMC *SMCCallerSession) GetNotaryInCommittee(_shardId *big.Int) (common.Address, error) {
	return _SMC.Contract.GetNotaryInCommittee(&_SMC.CallOpts, _shardId)
}

// GetVoteCount is a free data retrieval call binding the contract method 0xb2c2f2e8.
//
// Solidity: function getVoteCount(_shardId uint256) constant returns(uint256)
func (_SMC *SMCCaller) GetVoteCount(opts *bind.CallOpts, _shardId *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "getVoteCount", _shardId)
	return *ret0, err
}

// GetVoteCount is a free data retrieval call binding the contract method 0xb2c2f2e8.
//
// Solidity: function getVoteCount(_shardId uint256) constant returns(uint256)
func (_SMC *SMCSession) GetVoteCount(_shardId *big.Int) (*big.Int, error) {
	return _SMC.Contract.GetVoteCount(&_SMC.CallOpts, _shardId)
}

// GetVoteCount is a free data retrieval call binding the contract method 0xb2c2f2e8.
//
// Solidity: function getVoteCount(_shardId uint256) constant returns(uint256)
func (_SMC *SMCCallerSession) GetVoteCount(_shardId *big.Int) (*big.Int, error) {
	return _SMC.Contract.GetVoteCount(&_SMC.CallOpts, _shardId)
}

// HasVoted is a free data retrieval call binding the contract method 0x64390ff1.
//
// Solidity: function hasVoted(_shardId uint256, _index uint256) constant returns(bool)
func (_SMC *SMCCaller) HasVoted(opts *bind.CallOpts, _shardId *big.Int, _index *big.Int) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "hasVoted", _shardId, _index)
	return *ret0, err
}

// HasVoted is a free data retrieval call binding the contract method 0x64390ff1.
//
// Solidity: function hasVoted(_shardId uint256, _index uint256) constant returns(bool)
func (_SMC *SMCSession) HasVoted(_shardId *big.Int, _index *big.Int) (bool, error) {
	return _SMC.Contract.HasVoted(&_SMC.CallOpts, _shardId, _index)
}

// HasVoted is a free data retrieval call binding the contract method 0x64390ff1.
//
// Solidity: function hasVoted(_shardId uint256, _index uint256) constant returns(bool)
func (_SMC *SMCCallerSession) HasVoted(_shardId *big.Int, _index *big.Int) (bool, error) {
	return _SMC.Contract.HasVoted(&_SMC.CallOpts, _shardId, _index)
}

// LastApprovedCollation is a free data retrieval call binding the contract method 0x97d369a2.
//
// Solidity: function lastApprovedCollation( uint256) constant returns(uint256)
func (_SMC *SMCCaller) LastApprovedCollation(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "lastApprovedCollation", arg0)
	return *ret0, err
}

// LastApprovedCollation is a free data retrieval call binding the contract method 0x97d369a2.
//
// Solidity: function lastApprovedCollation( uint256) constant returns(uint256)
func (_SMC *SMCSession) LastApprovedCollation(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.LastApprovedCollation(&_SMC.CallOpts, arg0)
}

// LastApprovedCollation is a free data retrieval call binding the contract method 0x97d369a2.
//
// Solidity: function lastApprovedCollation( uint256) constant returns(uint256)
func (_SMC *SMCCallerSession) LastApprovedCollation(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.LastApprovedCollation(&_SMC.CallOpts, arg0)
}

// LastSubmittedCollation is a free data retrieval call binding the contract method 0x83ceeabe.
//
// Solidity: function lastSubmittedCollation( uint256) constant returns(uint256)
func (_SMC *SMCCaller) LastSubmittedCollation(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "lastSubmittedCollation", arg0)
	return *ret0, err
}

// LastSubmittedCollation is a free data retrieval call binding the contract method 0x83ceeabe.
//
// Solidity: function lastSubmittedCollation( uint256) constant returns(uint256)
func (_SMC *SMCSession) LastSubmittedCollation(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.LastSubmittedCollation(&_SMC.CallOpts, arg0)
}

// LastSubmittedCollation is a free data retrieval call binding the contract method 0x83ceeabe.
//
// Solidity: function lastSubmittedCollation( uint256) constant returns(uint256)
func (_SMC *SMCCallerSession) LastSubmittedCollation(arg0 *big.Int) (*big.Int, error) {
	return _SMC.Contract.LastSubmittedCollation(&_SMC.CallOpts, arg0)
}

// NotaryPool is a free data retrieval call binding the contract method 0xa81f4510.
//
// Solidity: function notaryPool( uint256) constant returns(address)
func (_SMC *SMCCaller) NotaryPool(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "notaryPool", arg0)
	return *ret0, err
}

// NotaryPool is a free data retrieval call binding the contract method 0xa81f4510.
//
// Solidity: function notaryPool( uint256) constant returns(address)
func (_SMC *SMCSession) NotaryPool(arg0 *big.Int) (common.Address, error) {
	return _SMC.Contract.NotaryPool(&_SMC.CallOpts, arg0)
}

// NotaryPool is a free data retrieval call binding the contract method 0xa81f4510.
//
// Solidity: function notaryPool( uint256) constant returns(address)
func (_SMC *SMCCallerSession) NotaryPool(arg0 *big.Int) (common.Address, error) {
	return _SMC.Contract.NotaryPool(&_SMC.CallOpts, arg0)
}

// NotaryPoolLength is a free data retrieval call binding the contract method 0xf6f67d36.
//
// Solidity: function notaryPoolLength() constant returns(uint256)
func (_SMC *SMCCaller) NotaryPoolLength(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "notaryPoolLength")
	return *ret0, err
}

// NotaryPoolLength is a free data retrieval call binding the contract method 0xf6f67d36.
//
// Solidity: function notaryPoolLength() constant returns(uint256)
func (_SMC *SMCSession) NotaryPoolLength() (*big.Int, error) {
	return _SMC.Contract.NotaryPoolLength(&_SMC.CallOpts)
}

// NotaryPoolLength is a free data retrieval call binding the contract method 0xf6f67d36.
//
// Solidity: function notaryPoolLength() constant returns(uint256)
func (_SMC *SMCCallerSession) NotaryPoolLength() (*big.Int, error) {
	return _SMC.Contract.NotaryPoolLength(&_SMC.CallOpts)
}

// NotaryRegistry is a free data retrieval call binding the contract method 0x6bdd3271.
//
// Solidity: function notaryRegistry( address) constant returns(deregisteredPeriod uint256, poolIndex uint256, balance uint256, deposited bool)
func (_SMC *SMCCaller) NotaryRegistry(opts *bind.CallOpts, arg0 common.Address) (struct {
	DeregisteredPeriod *big.Int
	PoolIndex          *big.Int
	Balance            *big.Int
	Deposited          bool
}, error) {
	ret := new(struct {
		DeregisteredPeriod *big.Int
		PoolIndex          *big.Int
		Balance            *big.Int
		Deposited          bool
	})
	out := ret
	err := _SMC.contract.Call(opts, out, "notaryRegistry", arg0)
	return *ret, err
}

// NotaryRegistry is a free data retrieval call binding the contract method 0x6bdd3271.
//
// Solidity: function notaryRegistry( address) constant returns(deregisteredPeriod uint256, poolIndex uint256, balance uint256, deposited bool)
func (_SMC *SMCSession) NotaryRegistry(arg0 common.Address) (struct {
	DeregisteredPeriod *big.Int
	PoolIndex          *big.Int
	Balance            *big.Int
	Deposited          bool
}, error) {
	return _SMC.Contract.NotaryRegistry(&_SMC.CallOpts, arg0)
}

// NotaryRegistry is a free data retrieval call binding the contract method 0x6bdd3271.
//
// Solidity: function notaryRegistry( address) constant returns(deregisteredPeriod uint256, poolIndex uint256, balance uint256, deposited bool)
func (_SMC *SMCCallerSession) NotaryRegistry(arg0 common.Address) (struct {
	DeregisteredPeriod *big.Int
	PoolIndex          *big.Int
	Balance            *big.Int
	Deposited          bool
}, error) {
	return _SMC.Contract.NotaryRegistry(&_SMC.CallOpts, arg0)
}

// ShardCount is a free data retrieval call binding the contract method 0x04e9c77a.
//
// Solidity: function shardCount() constant returns(uint256)
func (_SMC *SMCCaller) ShardCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "shardCount")
	return *ret0, err
}

// ShardCount is a free data retrieval call binding the contract method 0x04e9c77a.
//
// Solidity: function shardCount() constant returns(uint256)
func (_SMC *SMCSession) ShardCount() (*big.Int, error) {
	return _SMC.Contract.ShardCount(&_SMC.CallOpts)
}

// ShardCount is a free data retrieval call binding the contract method 0x04e9c77a.
//
// Solidity: function shardCount() constant returns(uint256)
func (_SMC *SMCCallerSession) ShardCount() (*big.Int, error) {
	return _SMC.Contract.ShardCount(&_SMC.CallOpts)
}

// AddHeader is a paid mutator transaction binding the contract method 0x75bd9912.
//
// Solidity: function addHeader(_shardId uint256, _period uint256, _chunkRoot bytes32) returns()
func (_SMC *SMCTransactor) AddHeader(opts *bind.TransactOpts, _shardId *big.Int, _period *big.Int, _chunkRoot [32]byte) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "addHeader", _shardId, _period, _chunkRoot)
}

// AddHeader is a paid mutator transaction binding the contract method 0x75bd9912.
//
// Solidity: function addHeader(_shardId uint256, _period uint256, _chunkRoot bytes32) returns()
func (_SMC *SMCSession) AddHeader(_shardId *big.Int, _period *big.Int, _chunkRoot [32]byte) (*types.Transaction, error) {
	return _SMC.Contract.AddHeader(&_SMC.TransactOpts, _shardId, _period, _chunkRoot)
}

// AddHeader is a paid mutator transaction binding the contract method 0x75bd9912.
//
// Solidity: function addHeader(_shardId uint256, _period uint256, _chunkRoot bytes32) returns()
func (_SMC *SMCTransactorSession) AddHeader(_shardId *big.Int, _period *big.Int, _chunkRoot [32]byte) (*types.Transaction, error) {
	return _SMC.Contract.AddHeader(&_SMC.TransactOpts, _shardId, _period, _chunkRoot)
}

// DeregisterNotary is a paid mutator transaction binding the contract method 0x58377bd1.
//
// Solidity: function deregisterNotary() returns()
func (_SMC *SMCTransactor) DeregisterNotary(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "deregisterNotary")
}

// DeregisterNotary is a paid mutator transaction binding the contract method 0x58377bd1.
//
// Solidity: function deregisterNotary() returns()
func (_SMC *SMCSession) DeregisterNotary() (*types.Transaction, error) {
	return _SMC.Contract.DeregisterNotary(&_SMC.TransactOpts)
}

// DeregisterNotary is a paid mutator transaction binding the contract method 0x58377bd1.
//
// Solidity: function deregisterNotary() returns()
func (_SMC *SMCTransactorSession) DeregisterNotary() (*types.Transaction, error) {
	return _SMC.Contract.DeregisterNotary(&_SMC.TransactOpts)
}

// RegisterNotary is a paid mutator transaction binding the contract method 0x68e9513e.
//
// Solidity: function registerNotary() returns()
func (_SMC *SMCTransactor) RegisterNotary(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "registerNotary")
}

// RegisterNotary is a paid mutator transaction binding the contract method 0x68e9513e.
//
// Solidity: function registerNotary() returns()
func (_SMC *SMCSession) RegisterNotary() (*types.Transaction, error) {
	return _SMC.Contract.RegisterNotary(&_SMC.TransactOpts)
}

// RegisterNotary is a paid mutator transaction binding the contract method 0x68e9513e.
//
// Solidity: function registerNotary() returns()
func (_SMC *SMCTransactorSession) RegisterNotary() (*types.Transaction, error) {
	return _SMC.Contract.RegisterNotary(&_SMC.TransactOpts)
}

// ReleaseNotary is a paid mutator transaction binding the contract method 0x9910851d.
//
// Solidity: function releaseNotary() returns()
func (_SMC *SMCTransactor) ReleaseNotary(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "releaseNotary")
}

// ReleaseNotary is a paid mutator transaction binding the contract method 0x9910851d.
//
// Solidity: function releaseNotary() returns()
func (_SMC *SMCSession) ReleaseNotary() (*types.Transaction, error) {
	return _SMC.Contract.ReleaseNotary(&_SMC.TransactOpts)
}

// ReleaseNotary is a paid mutator transaction binding the contract method 0x9910851d.
//
// Solidity: function releaseNotary() returns()
func (_SMC *SMCTransactorSession) ReleaseNotary() (*types.Transaction, error) {
	return _SMC.Contract.ReleaseNotary(&_SMC.TransactOpts)
}

// SubmitVote is a paid mutator transaction binding the contract method 0x4f33ffa0.
//
// Solidity: function submitVote(_shardId uint256, _period uint256, _index uint256, _chunkRoot bytes32) returns()
func (_SMC *SMCTransactor) SubmitVote(opts *bind.TransactOpts, _shardId *big.Int, _period *big.Int, _index *big.Int, _chunkRoot [32]byte) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "submitVote", _shardId, _period, _index, _chunkRoot)
}

// SubmitVote is a paid mutator transaction binding the contract method 0x4f33ffa0.
//
// Solidity: function submitVote(_shardId uint256, _period uint256, _index uint256, _chunkRoot bytes32) returns()
func (_SMC *SMCSession) SubmitVote(_shardId *big.Int, _period *big.Int, _index *big.Int, _chunkRoot [32]byte) (*types.Transaction, error) {
	return _SMC.Contract.SubmitVote(&_SMC.TransactOpts, _shardId, _period, _index, _chunkRoot)
}

// SubmitVote is a paid mutator transaction binding the contract method 0x4f33ffa0.
//
// Solidity: function submitVote(_shardId uint256, _period uint256, _index uint256, _chunkRoot bytes32) returns()
func (_SMC *SMCTransactorSession) SubmitVote(_shardId *big.Int, _period *big.Int, _index *big.Int, _chunkRoot [32]byte) (*types.Transaction, error) {
	return _SMC.Contract.SubmitVote(&_SMC.TransactOpts, _shardId, _period, _index, _chunkRoot)
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
	ShardId         *big.Int
	ChunkRoot       [32]byte
	Period          *big.Int
	ProposerAddress common.Address
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterHeaderAdded is a free log retrieval operation binding the contract event 0x2d0a86178d2fd307b47be157a766e6bee19bc26161c32f9781ee0e818636f09c.
//
// Solidity: e HeaderAdded(shardId indexed uint256, chunkRoot bytes32, period uint256, proposerAddress address)
func (_SMC *SMCFilterer) FilterHeaderAdded(opts *bind.FilterOpts, shardId []*big.Int) (*SMCHeaderAddedIterator, error) {

	var shardIdRule []interface{}
	for _, shardIdItem := range shardId {
		shardIdRule = append(shardIdRule, shardIdItem)
	}

	logs, sub, err := _SMC.contract.FilterLogs(opts, "HeaderAdded", shardIdRule)
	if err != nil {
		return nil, err
	}
	return &SMCHeaderAddedIterator{contract: _SMC.contract, event: "HeaderAdded", logs: logs, sub: sub}, nil
}

// WatchHeaderAdded is a free log subscription operation binding the contract event 0x2d0a86178d2fd307b47be157a766e6bee19bc26161c32f9781ee0e818636f09c.
//
// Solidity: e HeaderAdded(shardId indexed uint256, chunkRoot bytes32, period uint256, proposerAddress address)
func (_SMC *SMCFilterer) WatchHeaderAdded(opts *bind.WatchOpts, sink chan<- *SMCHeaderAdded, shardId []*big.Int) (event.Subscription, error) {

	var shardIdRule []interface{}
	for _, shardIdItem := range shardId {
		shardIdRule = append(shardIdRule, shardIdItem)
	}

	logs, sub, err := _SMC.contract.WatchLogs(opts, "HeaderAdded", shardIdRule)
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

// SMCNotaryDeregisteredIterator is returned from FilterNotaryDeregistered and is used to iterate over the raw logs and unpacked data for NotaryDeregistered events raised by the SMC contract.
type SMCNotaryDeregisteredIterator struct {
	Event *SMCNotaryDeregistered // Event containing the contract specifics and raw log

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
func (it *SMCNotaryDeregisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCNotaryDeregistered)
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
		it.Event = new(SMCNotaryDeregistered)
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
func (it *SMCNotaryDeregisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCNotaryDeregisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCNotaryDeregistered represents a NotaryDeregistered event raised by the SMC contract.
type SMCNotaryDeregistered struct {
	Notary             common.Address
	PoolIndex          *big.Int
	DeregisteredPeriod *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterNotaryDeregistered is a free log retrieval operation binding the contract event 0x90e5afdc8fd31453dcf6e37154fa117ddf3b0324c96c65015563df9d5e4b5a75.
//
// Solidity: e NotaryDeregistered(notary address, poolIndex uint256, deregisteredPeriod uint256)
func (_SMC *SMCFilterer) FilterNotaryDeregistered(opts *bind.FilterOpts) (*SMCNotaryDeregisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "NotaryDeregistered")
	if err != nil {
		return nil, err
	}
	return &SMCNotaryDeregisteredIterator{contract: _SMC.contract, event: "NotaryDeregistered", logs: logs, sub: sub}, nil
}

// WatchNotaryDeregistered is a free log subscription operation binding the contract event 0x90e5afdc8fd31453dcf6e37154fa117ddf3b0324c96c65015563df9d5e4b5a75.
//
// Solidity: e NotaryDeregistered(notary address, poolIndex uint256, deregisteredPeriod uint256)
func (_SMC *SMCFilterer) WatchNotaryDeregistered(opts *bind.WatchOpts, sink chan<- *SMCNotaryDeregistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "NotaryDeregistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCNotaryDeregistered)
				if err := _SMC.contract.UnpackLog(event, "NotaryDeregistered", log); err != nil {
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

// SMCNotaryRegisteredIterator is returned from FilterNotaryRegistered and is used to iterate over the raw logs and unpacked data for NotaryRegistered events raised by the SMC contract.
type SMCNotaryRegisteredIterator struct {
	Event *SMCNotaryRegistered // Event containing the contract specifics and raw log

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
func (it *SMCNotaryRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCNotaryRegistered)
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
		it.Event = new(SMCNotaryRegistered)
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
func (it *SMCNotaryRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCNotaryRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCNotaryRegistered represents a NotaryRegistered event raised by the SMC contract.
type SMCNotaryRegistered struct {
	Notary    common.Address
	PoolIndex *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterNotaryRegistered is a free log retrieval operation binding the contract event 0xa4fe15c53db34d35a5117acc26c27a2653dc68e2dadfc21ed211e38b7864d7a7.
//
// Solidity: e NotaryRegistered(notary address, poolIndex uint256)
func (_SMC *SMCFilterer) FilterNotaryRegistered(opts *bind.FilterOpts) (*SMCNotaryRegisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "NotaryRegistered")
	if err != nil {
		return nil, err
	}
	return &SMCNotaryRegisteredIterator{contract: _SMC.contract, event: "NotaryRegistered", logs: logs, sub: sub}, nil
}

// WatchNotaryRegistered is a free log subscription operation binding the contract event 0xa4fe15c53db34d35a5117acc26c27a2653dc68e2dadfc21ed211e38b7864d7a7.
//
// Solidity: e NotaryRegistered(notary address, poolIndex uint256)
func (_SMC *SMCFilterer) WatchNotaryRegistered(opts *bind.WatchOpts, sink chan<- *SMCNotaryRegistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "NotaryRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCNotaryRegistered)
				if err := _SMC.contract.UnpackLog(event, "NotaryRegistered", log); err != nil {
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

// SMCNotaryReleasedIterator is returned from FilterNotaryReleased and is used to iterate over the raw logs and unpacked data for NotaryReleased events raised by the SMC contract.
type SMCNotaryReleasedIterator struct {
	Event *SMCNotaryReleased // Event containing the contract specifics and raw log

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
func (it *SMCNotaryReleasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCNotaryReleased)
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
		it.Event = new(SMCNotaryReleased)
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
func (it *SMCNotaryReleasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCNotaryReleasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCNotaryReleased represents a NotaryReleased event raised by the SMC contract.
type SMCNotaryReleased struct {
	Notary    common.Address
	PoolIndex *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterNotaryReleased is a free log retrieval operation binding the contract event 0xaee20171b64b7f3360a142659094ce929970d6963dcea8c34a9bf1ece8033680.
//
// Solidity: e NotaryReleased(notary address, poolIndex uint256)
func (_SMC *SMCFilterer) FilterNotaryReleased(opts *bind.FilterOpts) (*SMCNotaryReleasedIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "NotaryReleased")
	if err != nil {
		return nil, err
	}
	return &SMCNotaryReleasedIterator{contract: _SMC.contract, event: "NotaryReleased", logs: logs, sub: sub}, nil
}

// WatchNotaryReleased is a free log subscription operation binding the contract event 0xaee20171b64b7f3360a142659094ce929970d6963dcea8c34a9bf1ece8033680.
//
// Solidity: e NotaryReleased(notary address, poolIndex uint256)
func (_SMC *SMCFilterer) WatchNotaryReleased(opts *bind.WatchOpts, sink chan<- *SMCNotaryReleased) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "NotaryReleased")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCNotaryReleased)
				if err := _SMC.contract.UnpackLog(event, "NotaryReleased", log); err != nil {
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

// SMCVoteSubmittedIterator is returned from FilterVoteSubmitted and is used to iterate over the raw logs and unpacked data for VoteSubmitted events raised by the SMC contract.
type SMCVoteSubmittedIterator struct {
	Event *SMCVoteSubmitted // Event containing the contract specifics and raw log

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
func (it *SMCVoteSubmittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCVoteSubmitted)
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
		it.Event = new(SMCVoteSubmitted)
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
func (it *SMCVoteSubmittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCVoteSubmittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCVoteSubmitted represents a VoteSubmitted event raised by the SMC contract.
type SMCVoteSubmitted struct {
	ShardId       *big.Int
	ChunkRoot     [32]byte
	Period        *big.Int
	NotaryAddress common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterVoteSubmitted is a free log retrieval operation binding the contract event 0xc99370212b708f699fb6945a17eb34d0fc1ccd5b45d88f4d9682593a45d6e833.
//
// Solidity: e VoteSubmitted(shardId indexed uint256, chunkRoot bytes32, period uint256, notaryAddress address)
func (_SMC *SMCFilterer) FilterVoteSubmitted(opts *bind.FilterOpts, shardId []*big.Int) (*SMCVoteSubmittedIterator, error) {

	var shardIdRule []interface{}
	for _, shardIdItem := range shardId {
		shardIdRule = append(shardIdRule, shardIdItem)
	}

	logs, sub, err := _SMC.contract.FilterLogs(opts, "VoteSubmitted", shardIdRule)
	if err != nil {
		return nil, err
	}
	return &SMCVoteSubmittedIterator{contract: _SMC.contract, event: "VoteSubmitted", logs: logs, sub: sub}, nil
}

// WatchVoteSubmitted is a free log subscription operation binding the contract event 0xc99370212b708f699fb6945a17eb34d0fc1ccd5b45d88f4d9682593a45d6e833.
//
// Solidity: e VoteSubmitted(shardId indexed uint256, chunkRoot bytes32, period uint256, notaryAddress address)
func (_SMC *SMCFilterer) WatchVoteSubmitted(opts *bind.WatchOpts, sink chan<- *SMCVoteSubmitted, shardId []*big.Int) (event.Subscription, error) {

	var shardIdRule []interface{}
	for _, shardIdItem := range shardId {
		shardIdRule = append(shardIdRule, shardIdItem)
	}

	logs, sub, err := _SMC.contract.WatchLogs(opts, "VoteSubmitted", shardIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCVoteSubmitted)
				if err := _SMC.contract.UnpackLog(event, "VoteSubmitted", log); err != nil {
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
