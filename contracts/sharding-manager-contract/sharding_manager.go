// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package smc

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

// SMCABI is the input ABI used to generate the binding from.
const SMCABI = "[{\"constant\":false,\"inputs\":[],\"name\":\"releaseAttester\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"shardCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"deregisterAttester\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"currentVote\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"attesterPoolLength\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"},{\"name\":\"_period\",\"type\":\"uint256\"},{\"name\":\"_index\",\"type\":\"uint256\"},{\"name\":\"_chunkRoot\",\"type\":\"bytes32\"}],\"name\":\"submitVote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"registerAttester\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"},{\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"hasVoted\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"}],\"name\":\"getAttesterInCommittee\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"lastSubmittedCollation\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"lastApprovedCollation\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"attesterPool\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"}],\"name\":\"getVoteCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"CHALLENGE_PERIOD\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"uint256\"},{\"name\":\"_period\",\"type\":\"uint256\"},{\"name\":\"_chunkRoot\",\"type\":\"bytes32\"},{\"name\":\"_signature\",\"type\":\"bytes32\"}],\"name\":\"addHeader\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"collationRecords\",\"outputs\":[{\"name\":\"chunkRoot\",\"type\":\"bytes32\"},{\"name\":\"proposer\",\"type\":\"address\"},{\"name\":\"isElected\",\"type\":\"bool\"},{\"name\":\"signature\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"attesterRegistry\",\"outputs\":[{\"name\":\"deregisteredPeriod\",\"type\":\"uint256\"},{\"name\":\"poolIndex\",\"type\":\"uint256\"},{\"name\":\"balance\",\"type\":\"uint256\"},{\"name\":\"deposited\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"shardId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"chunkRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"period\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"proposerAddress\",\"type\":\"address\"}],\"name\":\"HeaderAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"poolIndex\",\"type\":\"uint256\"}],\"name\":\"AttesterRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"poolIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"deregisteredPeriod\",\"type\":\"uint256\"}],\"name\":\"AttesterDeregistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"poolIndex\",\"type\":\"uint256\"}],\"name\":\"AttesterReleased\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"shardId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"chunkRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"period\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"attesterAddress\",\"type\":\"address\"}],\"name\":\"VoteSubmitted\",\"type\":\"event\"}]"

// SMCBin is the compiled bytecode used for deploying new contracts.
const SMCBin = `0x60806040526064600c5534801561001557600080fd5b50610d58806100256000396000f3006080604052600436106100f05763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630340f4bf81146100f557806304e9c77a1461010c57806307f9ccd1146101335780630c8da4cc14610148578063375246d9146101605780634f33ffa0146101755780634f81e1191461019657806364390ff11461019e5780637d6c5302146101cd57806383ceeabe1461020157806397d369a214610219578063a1943b4214610231578063b2c2f2e814610249578063c3a079ed14610261578063c4d5f19814610276578063e9e0b68314610297578063fd2943da146102e0575b600080fd5b34801561010157600080fd5b5061010a610329565b005b34801561011857600080fd5b5061012161045e565b60408051918252519081900360200190f35b34801561013f57600080fd5b5061010a610464565b34801561015457600080fd5b5061012160043561058f565b34801561016c57600080fd5b506101216105a1565b34801561018157600080fd5b5061010a6004356024356044356064356105a7565b61010a61074d565b3480156101aa57600080fd5b506101b9600435602435610910565b604080519115158252519081900360200190f35b3480156101d957600080fd5b506101e5600435610933565b60408051600160a060020a039092168252519081900360200190f35b34801561020d57600080fd5b506101216004356109ec565b34801561022557600080fd5b506101216004356109fe565b34801561023d57600080fd5b506101e5600435610a10565b34801561025557600080fd5b50610121600435610a38565b34801561026d57600080fd5b50610121610a4d565b34801561028257600080fd5b5061010a600435602435604435606435610a52565b3480156102a357600080fd5b506102b2600435602435610bb1565b60408051948552600160a060020a039093166020850152901515838301526060830152519081900360800190f35b3480156102ec57600080fd5b50610301600160a060020a0360043516610c04565b6040805194855260208501939093528383019190915215156060830152519081900360800190f35b33600160a060020a038116600090815260016020819052604082208082015460039091015490929160ff90911615151461036257600080fd5b600160a060020a038316600090815260016020526040902054151561038657600080fd5b600160a060020a038316600090815260016020526040902054613f000160054304116103b157600080fd5b50600160a060020a0382166000818152600160208190526040808320600281018054858355938201859055849055600301805460ff191690555190929183156108fc02918491818181858888f19350505050158015610414573d6000803e3d6000fd5b5060408051600160a060020a03851681526020810184905281517f0953df78e93a11708482200aba752831b653dcd4bc029159b6830e8e5e1099fc929181900390910190a1505050565b600c5481565b33600160a060020a038116600090815260016020819052604082209081015460039091015490919060ff16151561049a57600080fd5b82600160a060020a03166000838154811015156104b357fe5b600091825260209091200154600160a060020a0316146104d257600080fd5b6104da610c2f565b50600160a060020a0382166000908152600160205260409020600543049081905561050482610c52565b600080548390811061051257fe5b600091825260209182902001805473ffffffffffffffffffffffffffffffffffffffff191690556002805460001901905560408051600160a060020a0386168152918201849052818101839052517fd7731db8678a142362195505e228a3625ff64ce27c12ff97dd5c4b859a2346c19181900360600190a1505050565b60036020526000908152604090205481565b60025481565b60008085101580156105ba5750600c5485105b15156105c557600080fd5b6005430484146105d457600080fd5b60008581526005602052604090205484146105ee57600080fd5b608783106105fb57600080fd5b6000858152600460209081526040808320878452909152902054821461062057600080fd5b600160a060020a03331660009081526001602052604090206003015460ff16151561064a57600080fd5b6106548584610910565b1561065e57600080fd5b33600160a060020a031661067186610933565b600160a060020a03161461068457600080fd5b61068e8584610cc3565b61069785610a38565b9050605a81106106fb576000858152600660209081526040808320879055600482528083208784529091529020600101805474ff00000000000000000000000000000000000000001916740100000000000000000000000000000000000000001790555b6040805183815260208101869052600160a060020a03331681830152905186917fc99370212b708f699fb6945a17eb34d0fc1ccd5b45d88f4d9682593a45d6e833919081900360600190a25050505050565b33600160a060020a03811660009081526001602052604081206003015460ff161561077757600080fd5b34683635c9adc5dea000001461078c57600080fd5b610794610c2f565b61079c610ce7565b156107ff5750600254600080546001810182559080527f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56301805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a038416179055610848565b610807610cee565b90508160008281548110151561081957fe5b9060005260206000200160006101000a815481600160a060020a030219169083600160a060020a031602179055505b600280546001908101825560408051608081018252600080825260208083018781523484860190815260608501878152600160a060020a038b168552928790529490922092518355905193820193909355905192810192909255516003909101805460ff1916911515919091179055600a5481106108c85760018101600a555b60408051600160a060020a03841681526020810183905281517f9e36cf4a00da2cfda2c9456ba639ee573dbdf53e4487daf51b66d98232d726cc929181900390910190a15050565b60009182526003602052604090912054600160ff9290920360020a900481161490565b6000600543048180808080610946610c2f565b600b5486111561095a57600a549450610960565b60095494505b600160a060020a03331660009081526001602081815260409283902090910154825160001960058b020180408083529382018390528185018d90529351908190036060019020909650919450925085908115156109b957fe5b0690506000818154811015156109cb57fe5b600091825260209091200154600160a060020a031698975050505050505050565b60056020526000908152604090205481565b60066020526000908152604090205481565b6000805482908110610a1e57fe5b600091825260209091200154600160a060020a0316905081565b60009081526003602052604090205460ff1690565b601981565b60008410158015610a645750600c5484105b1515610a6f57600080fd5b600543048314610a7e57600080fd5b6000848152600560205260409020548311610a9857600080fd5b610aa0610c2f565b60408051608081018252838152600160a060020a033381166020808401828152600085870181815260608088018a81528d8452600486528984208d8552865289842098518955935160018901805493511515740100000000000000000000000000000000000000000274ff0000000000000000000000000000000000000000199290991673ffffffffffffffffffffffffffffffffffffffff19909416939093171696909617905590516002909501949094558884526005808252858520439190910490556003815284842093909355835186815292830187905282840152915186927f2d0a86178d2fd307b47be157a766e6bee19bc26161c32f9781ee0e818636f09c928290030190a250505050565b60046020908152600092835260408084209091529082529020805460018201546002909201549091600160a060020a038116917401000000000000000000000000000000000000000090910460ff169084565b6001602081905260009182526040909120805491810154600282015460039092015490919060ff1684565b600b546005430490811015610c4357610c4f565b600a54600955600b8190555b50565b6008546007541415610c9857600780546001810182556000919091527fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c68801819055610cb7565b806007600854815481101515610caa57fe5b6000918252602090912001555b50600880546001019055565b600091825260036020526040909120805460ff9290920360020a9091176001019055565b6008541590565b60006001600854111515610d0157600080fd5b600880546000190190819055600780549091908110610d1c57fe5b90600052602060002001549050905600a165627a7a72305820f007be423442b9c3f61cfce458bd20c8efcf2953ba5ab82c06da0dd2bb641e550029`

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

// AttesterPool is a free data retrieval call binding the contract method 0xa1943b42.
//
// Solidity: function attesterPool( uint256) constant returns(address)
func (_SMC *SMCCaller) AttesterPool(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "attesterPool", arg0)
	return *ret0, err
}

// AttesterPool is a free data retrieval call binding the contract method 0xa1943b42.
//
// Solidity: function attesterPool( uint256) constant returns(address)
func (_SMC *SMCSession) AttesterPool(arg0 *big.Int) (common.Address, error) {
	return _SMC.Contract.AttesterPool(&_SMC.CallOpts, arg0)
}

// AttesterPool is a free data retrieval call binding the contract method 0xa1943b42.
//
// Solidity: function attesterPool( uint256) constant returns(address)
func (_SMC *SMCCallerSession) AttesterPool(arg0 *big.Int) (common.Address, error) {
	return _SMC.Contract.AttesterPool(&_SMC.CallOpts, arg0)
}

// AttesterPoolLength is a free data retrieval call binding the contract method 0x375246d9.
//
// Solidity: function attesterPoolLength() constant returns(uint256)
func (_SMC *SMCCaller) AttesterPoolLength(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "attesterPoolLength")
	return *ret0, err
}

// AttesterPoolLength is a free data retrieval call binding the contract method 0x375246d9.
//
// Solidity: function attesterPoolLength() constant returns(uint256)
func (_SMC *SMCSession) AttesterPoolLength() (*big.Int, error) {
	return _SMC.Contract.AttesterPoolLength(&_SMC.CallOpts)
}

// AttesterPoolLength is a free data retrieval call binding the contract method 0x375246d9.
//
// Solidity: function attesterPoolLength() constant returns(uint256)
func (_SMC *SMCCallerSession) AttesterPoolLength() (*big.Int, error) {
	return _SMC.Contract.AttesterPoolLength(&_SMC.CallOpts)
}

// AttesterRegistry is a free data retrieval call binding the contract method 0xfd2943da.
//
// Solidity: function attesterRegistry( address) constant returns(deregisteredPeriod uint256, poolIndex uint256, balance uint256, deposited bool)
func (_SMC *SMCCaller) AttesterRegistry(opts *bind.CallOpts, arg0 common.Address) (struct {
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
	err := _SMC.contract.Call(opts, out, "attesterRegistry", arg0)
	return *ret, err
}

// AttesterRegistry is a free data retrieval call binding the contract method 0xfd2943da.
//
// Solidity: function attesterRegistry( address) constant returns(deregisteredPeriod uint256, poolIndex uint256, balance uint256, deposited bool)
func (_SMC *SMCSession) AttesterRegistry(arg0 common.Address) (struct {
	DeregisteredPeriod *big.Int
	PoolIndex          *big.Int
	Balance            *big.Int
	Deposited          bool
}, error) {
	return _SMC.Contract.AttesterRegistry(&_SMC.CallOpts, arg0)
}

// AttesterRegistry is a free data retrieval call binding the contract method 0xfd2943da.
//
// Solidity: function attesterRegistry( address) constant returns(deregisteredPeriod uint256, poolIndex uint256, balance uint256, deposited bool)
func (_SMC *SMCCallerSession) AttesterRegistry(arg0 common.Address) (struct {
	DeregisteredPeriod *big.Int
	PoolIndex          *big.Int
	Balance            *big.Int
	Deposited          bool
}, error) {
	return _SMC.Contract.AttesterRegistry(&_SMC.CallOpts, arg0)
}

// CollationRecords is a free data retrieval call binding the contract method 0xe9e0b683.
//
// Solidity: function collationRecords( uint256,  uint256) constant returns(chunkRoot bytes32, proposer address, isElected bool, signature bytes32)
func (_SMC *SMCCaller) CollationRecords(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
	Signature [32]byte
}, error) {
	ret := new(struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
		Signature [32]byte
	})
	out := ret
	err := _SMC.contract.Call(opts, out, "collationRecords", arg0, arg1)
	return *ret, err
}

// CollationRecords is a free data retrieval call binding the contract method 0xe9e0b683.
//
// Solidity: function collationRecords( uint256,  uint256) constant returns(chunkRoot bytes32, proposer address, isElected bool, signature bytes32)
func (_SMC *SMCSession) CollationRecords(arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
	Signature [32]byte
}, error) {
	return _SMC.Contract.CollationRecords(&_SMC.CallOpts, arg0, arg1)
}

// CollationRecords is a free data retrieval call binding the contract method 0xe9e0b683.
//
// Solidity: function collationRecords( uint256,  uint256) constant returns(chunkRoot bytes32, proposer address, isElected bool, signature bytes32)
func (_SMC *SMCCallerSession) CollationRecords(arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
	Signature [32]byte
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

// GetAttesterInCommittee is a free data retrieval call binding the contract method 0x7d6c5302.
//
// Solidity: function getAttesterInCommittee(_shardId uint256) constant returns(address)
func (_SMC *SMCCaller) GetAttesterInCommittee(opts *bind.CallOpts, _shardId *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _SMC.contract.Call(opts, out, "getAttesterInCommittee", _shardId)
	return *ret0, err
}

// GetAttesterInCommittee is a free data retrieval call binding the contract method 0x7d6c5302.
//
// Solidity: function getAttesterInCommittee(_shardId uint256) constant returns(address)
func (_SMC *SMCSession) GetAttesterInCommittee(_shardId *big.Int) (common.Address, error) {
	return _SMC.Contract.GetAttesterInCommittee(&_SMC.CallOpts, _shardId)
}

// GetAttesterInCommittee is a free data retrieval call binding the contract method 0x7d6c5302.
//
// Solidity: function getAttesterInCommittee(_shardId uint256) constant returns(address)
func (_SMC *SMCCallerSession) GetAttesterInCommittee(_shardId *big.Int) (common.Address, error) {
	return _SMC.Contract.GetAttesterInCommittee(&_SMC.CallOpts, _shardId)
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

// AddHeader is a paid mutator transaction binding the contract method 0xc4d5f198.
//
// Solidity: function addHeader(_shardId uint256, _period uint256, _chunkRoot bytes32, _signature bytes32) returns()
func (_SMC *SMCTransactor) AddHeader(opts *bind.TransactOpts, _shardId *big.Int, _period *big.Int, _chunkRoot [32]byte, _signature [32]byte) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "addHeader", _shardId, _period, _chunkRoot, _signature)
}

// AddHeader is a paid mutator transaction binding the contract method 0xc4d5f198.
//
// Solidity: function addHeader(_shardId uint256, _period uint256, _chunkRoot bytes32, _signature bytes32) returns()
func (_SMC *SMCSession) AddHeader(_shardId *big.Int, _period *big.Int, _chunkRoot [32]byte, _signature [32]byte) (*types.Transaction, error) {
	return _SMC.Contract.AddHeader(&_SMC.TransactOpts, _shardId, _period, _chunkRoot, _signature)
}

// AddHeader is a paid mutator transaction binding the contract method 0xc4d5f198.
//
// Solidity: function addHeader(_shardId uint256, _period uint256, _chunkRoot bytes32, _signature bytes32) returns()
func (_SMC *SMCTransactorSession) AddHeader(_shardId *big.Int, _period *big.Int, _chunkRoot [32]byte, _signature [32]byte) (*types.Transaction, error) {
	return _SMC.Contract.AddHeader(&_SMC.TransactOpts, _shardId, _period, _chunkRoot, _signature)
}

// DeregisterAttester is a paid mutator transaction binding the contract method 0x07f9ccd1.
//
// Solidity: function deregisterAttester() returns()
func (_SMC *SMCTransactor) DeregisterAttester(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "deregisterAttester")
}

// DeregisterAttester is a paid mutator transaction binding the contract method 0x07f9ccd1.
//
// Solidity: function deregisterAttester() returns()
func (_SMC *SMCSession) DeregisterAttester() (*types.Transaction, error) {
	return _SMC.Contract.DeregisterAttester(&_SMC.TransactOpts)
}

// DeregisterAttester is a paid mutator transaction binding the contract method 0x07f9ccd1.
//
// Solidity: function deregisterAttester() returns()
func (_SMC *SMCTransactorSession) DeregisterAttester() (*types.Transaction, error) {
	return _SMC.Contract.DeregisterAttester(&_SMC.TransactOpts)
}

// RegisterAttester is a paid mutator transaction binding the contract method 0x4f81e119.
//
// Solidity: function registerAttester() returns()
func (_SMC *SMCTransactor) RegisterAttester(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "registerAttester")
}

// RegisterAttester is a paid mutator transaction binding the contract method 0x4f81e119.
//
// Solidity: function registerAttester() returns()
func (_SMC *SMCSession) RegisterAttester() (*types.Transaction, error) {
	return _SMC.Contract.RegisterAttester(&_SMC.TransactOpts)
}

// RegisterAttester is a paid mutator transaction binding the contract method 0x4f81e119.
//
// Solidity: function registerAttester() returns()
func (_SMC *SMCTransactorSession) RegisterAttester() (*types.Transaction, error) {
	return _SMC.Contract.RegisterAttester(&_SMC.TransactOpts)
}

// ReleaseAttester is a paid mutator transaction binding the contract method 0x0340f4bf.
//
// Solidity: function releaseAttester() returns()
func (_SMC *SMCTransactor) ReleaseAttester(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMC.contract.Transact(opts, "releaseAttester")
}

// ReleaseAttester is a paid mutator transaction binding the contract method 0x0340f4bf.
//
// Solidity: function releaseAttester() returns()
func (_SMC *SMCSession) ReleaseAttester() (*types.Transaction, error) {
	return _SMC.Contract.ReleaseAttester(&_SMC.TransactOpts)
}

// ReleaseAttester is a paid mutator transaction binding the contract method 0x0340f4bf.
//
// Solidity: function releaseAttester() returns()
func (_SMC *SMCTransactorSession) ReleaseAttester() (*types.Transaction, error) {
	return _SMC.Contract.ReleaseAttester(&_SMC.TransactOpts)
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

// SMCAttesterDeregisteredIterator is returned from FilterAttesterDeregistered and is used to iterate over the raw logs and unpacked data for AttesterDeregistered events raised by the SMC contract.
type SMCAttesterDeregisteredIterator struct {
	Event *SMCAttesterDeregistered // Event containing the contract specifics and raw log

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
func (it *SMCAttesterDeregisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCAttesterDeregistered)
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
		it.Event = new(SMCAttesterDeregistered)
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
func (it *SMCAttesterDeregisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCAttesterDeregisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCAttesterDeregistered represents a AttesterDeregistered event raised by the SMC contract.
type SMCAttesterDeregistered struct {
	Attester           common.Address
	PoolIndex          *big.Int
	DeregisteredPeriod *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterAttesterDeregistered is a free log retrieval operation binding the contract event 0xd7731db8678a142362195505e228a3625ff64ce27c12ff97dd5c4b859a2346c1.
//
// Solidity: event AttesterDeregistered(attester address, poolIndex uint256, deregisteredPeriod uint256)
func (_SMC *SMCFilterer) FilterAttesterDeregistered(opts *bind.FilterOpts) (*SMCAttesterDeregisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "AttesterDeregistered")
	if err != nil {
		return nil, err
	}
	return &SMCAttesterDeregisteredIterator{contract: _SMC.contract, event: "AttesterDeregistered", logs: logs, sub: sub}, nil
}

// WatchAttesterDeregistered is a free log subscription operation binding the contract event 0xd7731db8678a142362195505e228a3625ff64ce27c12ff97dd5c4b859a2346c1.
//
// Solidity: event AttesterDeregistered(attester address, poolIndex uint256, deregisteredPeriod uint256)
func (_SMC *SMCFilterer) WatchAttesterDeregistered(opts *bind.WatchOpts, sink chan<- *SMCAttesterDeregistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "AttesterDeregistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCAttesterDeregistered)
				if err := _SMC.contract.UnpackLog(event, "AttesterDeregistered", log); err != nil {
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

// SMCAttesterRegisteredIterator is returned from FilterAttesterRegistered and is used to iterate over the raw logs and unpacked data for AttesterRegistered events raised by the SMC contract.
type SMCAttesterRegisteredIterator struct {
	Event *SMCAttesterRegistered // Event containing the contract specifics and raw log

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
func (it *SMCAttesterRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCAttesterRegistered)
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
		it.Event = new(SMCAttesterRegistered)
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
func (it *SMCAttesterRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCAttesterRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCAttesterRegistered represents a AttesterRegistered event raised by the SMC contract.
type SMCAttesterRegistered struct {
	Attester  common.Address
	PoolIndex *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAttesterRegistered is a free log retrieval operation binding the contract event 0x9e36cf4a00da2cfda2c9456ba639ee573dbdf53e4487daf51b66d98232d726cc.
//
// Solidity: event AttesterRegistered(attester address, poolIndex uint256)
func (_SMC *SMCFilterer) FilterAttesterRegistered(opts *bind.FilterOpts) (*SMCAttesterRegisteredIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "AttesterRegistered")
	if err != nil {
		return nil, err
	}
	return &SMCAttesterRegisteredIterator{contract: _SMC.contract, event: "AttesterRegistered", logs: logs, sub: sub}, nil
}

// WatchAttesterRegistered is a free log subscription operation binding the contract event 0x9e36cf4a00da2cfda2c9456ba639ee573dbdf53e4487daf51b66d98232d726cc.
//
// Solidity: event AttesterRegistered(attester address, poolIndex uint256)
func (_SMC *SMCFilterer) WatchAttesterRegistered(opts *bind.WatchOpts, sink chan<- *SMCAttesterRegistered) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "AttesterRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCAttesterRegistered)
				if err := _SMC.contract.UnpackLog(event, "AttesterRegistered", log); err != nil {
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

// SMCAttesterReleasedIterator is returned from FilterAttesterReleased and is used to iterate over the raw logs and unpacked data for AttesterReleased events raised by the SMC contract.
type SMCAttesterReleasedIterator struct {
	Event *SMCAttesterReleased // Event containing the contract specifics and raw log

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
func (it *SMCAttesterReleasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMCAttesterReleased)
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
		it.Event = new(SMCAttesterReleased)
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
func (it *SMCAttesterReleasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMCAttesterReleasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMCAttesterReleased represents a AttesterReleased event raised by the SMC contract.
type SMCAttesterReleased struct {
	Attester  common.Address
	PoolIndex *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAttesterReleased is a free log retrieval operation binding the contract event 0x0953df78e93a11708482200aba752831b653dcd4bc029159b6830e8e5e1099fc.
//
// Solidity: event AttesterReleased(attester address, poolIndex uint256)
func (_SMC *SMCFilterer) FilterAttesterReleased(opts *bind.FilterOpts) (*SMCAttesterReleasedIterator, error) {

	logs, sub, err := _SMC.contract.FilterLogs(opts, "AttesterReleased")
	if err != nil {
		return nil, err
	}
	return &SMCAttesterReleasedIterator{contract: _SMC.contract, event: "AttesterReleased", logs: logs, sub: sub}, nil
}

// WatchAttesterReleased is a free log subscription operation binding the contract event 0x0953df78e93a11708482200aba752831b653dcd4bc029159b6830e8e5e1099fc.
//
// Solidity: event AttesterReleased(attester address, poolIndex uint256)
func (_SMC *SMCFilterer) WatchAttesterReleased(opts *bind.WatchOpts, sink chan<- *SMCAttesterReleased) (event.Subscription, error) {

	logs, sub, err := _SMC.contract.WatchLogs(opts, "AttesterReleased")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMCAttesterReleased)
				if err := _SMC.contract.UnpackLog(event, "AttesterReleased", log); err != nil {
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
	ShardId         *big.Int
	ChunkRoot       [32]byte
	Period          *big.Int
	ProposerAddress common.Address
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterHeaderAdded is a free log retrieval operation binding the contract event 0x2d0a86178d2fd307b47be157a766e6bee19bc26161c32f9781ee0e818636f09c.
//
// Solidity: event HeaderAdded(shardId indexed uint256, chunkRoot bytes32, period uint256, proposerAddress address)
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
// Solidity: event HeaderAdded(shardId indexed uint256, chunkRoot bytes32, period uint256, proposerAddress address)
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
	ShardId         *big.Int
	ChunkRoot       [32]byte
	Period          *big.Int
	AttesterAddress common.Address
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterVoteSubmitted is a free log retrieval operation binding the contract event 0xc99370212b708f699fb6945a17eb34d0fc1ccd5b45d88f4d9682593a45d6e833.
//
// Solidity: event VoteSubmitted(shardId indexed uint256, chunkRoot bytes32, period uint256, attesterAddress address)
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
// Solidity: event VoteSubmitted(shardId indexed uint256, chunkRoot bytes32, period uint256, attesterAddress address)
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
