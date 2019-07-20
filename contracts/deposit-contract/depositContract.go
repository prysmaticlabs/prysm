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
const DepositContractABI = "[{\"name\":\"DepositEvent\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"amount\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"signature\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"index\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"address\",\"name\":\"_drain_address\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"get_hash_tree_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":91734},{\"name\":\"get_deposit_count\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":10493},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\"},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\"},{\"type\":\"bytes\",\"name\":\"signature\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":1334707},{\"name\":\"drain\",\"outputs\":[],\"inputs\":[],\"constant\":false,\"payable\":false,\"type\":\"function\",\"gas\":35823},{\"name\":\"MIN_DEPOSIT_AMOUNT\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":663},{\"name\":\"deposit_count\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":693},{\"name\":\"drain_address\",\"outputs\":[{\"type\":\"address\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":723}]"

// DepositContractBin is the compiled bytecode used for deploying new contracts.
const DepositContractBin = `0x600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a05260406114ff6101403934156100a757600080fd5b602060206114ff0160c03960c05160205181106100c357600080fd5b5061014051600055610160516004556101806000601f818352015b600061018051602081106100f157600080fd5b600160c052602060c02001546020826101a0010152602081019050610180516020811061011d57600080fd5b600160c052602060c02001546020826101a0010152602081019050806101a0526101a09050602060c0825160208401600060025af161015b57600080fd5b60c0519050606051600161018051018060405190131561017a57600080fd5b809190121561018857600080fd5b6020811061019557600080fd5b600160c052602060c02001555b81516001018083528114156100de575b50506114e756600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a0526000156101a3575b6101605261014052601860086020820661018001602082840111156100bf57600080fd5b6020806101a082610140600060046015f1505081815280905090509050805160200180610240828460006004600a8704601201f16100fc57600080fd5b50506102405160206001820306601f82010390506102a0610240516008818352015b826102a051111561012e5761014a565b60006102a05161026001535b815160010180835281141561011e575b5050506020610220526040610240510160206001820306601f8201039050610200525b60006102005111151561017f5761019b565b602061020051036102200151602061020051036102005261016d565b610160515650005b600015610387575b6101605261014052600061018052610140516101a0526101c060006008818352015b61018051600860008112156101ea578060000360020a82046101f1565b8060020a82025b905090506101805260ff6101a051166101e052610180516101e0516101805101101561021c57600080fd5b6101e0516101805101610180526101a0517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff86000811215610265578060000360020a820461026c565b8060020a82025b905090506101a0525b81516001018083528114156101cd575b5050601860086020820661020001602082840111156102a357600080fd5b60208061022082610180600060046015f15050818152809050905090508051602001806102c0828460006004600a8704601201f16102e057600080fd5b50506102c05160206001820306601f82010390506103206102c0516008818352015b826103205111156103125761032e565b6000610320516102e001535b8151600101808352811415610302575b50505060206102a05260406102c0510160206001820306601f8201039050610280525b6000610280511115156103635761037f565b602061028051036102a001516020610280510361028052610351565b610160515650005b63863a311b60005114156106185734156103a057600080fd5b6000610140526101405161016052600354610180526101a060006020818352015b60016001610180511614156104425760006101a051602081106103e357600080fd5b600260c052602060c02001546020826102400101526020810190506101605160208261024001015260208101905080610240526102409050602060c0825160208401600060025af161043457600080fd5b60c0519050610160526104b0565b6000610160516020826101c00101526020810190506101a0516020811061046857600080fd5b600160c052602060c02001546020826101c0010152602081019050806101c0526101c09050602060c0825160208401600060025af16104a657600080fd5b60c0519050610160525b61018060026104be57600080fd5b60028151048152505b81516001018083528114156103c1575b505060006101605160208261044001015260208101905061014051610160516101805163806732896102c0526003546102e0526102e051600658016101ab565b506103405260006103a0525b6103405160206001820306601f82010390506103a0511015156105455761055e565b6103a05161036001526103a0516020016103a052610523565b61018052610160526101405261034060088060208461044001018260208501600060046012f150508051820191505060006018602082066103c001602082840111156105a957600080fd5b6020806103e082610140600060046015f150508181528090509050905060188060208461044001018260208501600060046014f150508051820191505080610440526104409050602060c0825160208401600060025af161060957600080fd5b60c051905060005260206000f3005b63621fd130600051141561072a57341561063157600080fd5b6380673289610140526003546101605261016051600658016101ab565b506101c0526000610220525b6101c05160206001820306601f82010390506102205110151561067c57610695565b610220516101e00152610220516020016102205261065a565b6101c0805160200180610280828460006004600a8704601201f16106b857600080fd5b50506102805160206001820306601f82010390506102e0610280516008818352015b826102e05111156106ea57610706565b60006102e0516102a001535b81516001018083528114156106da575b5050506020610260526040610280510160206001820306601f8201039050610260f3005b63c47e300d600051141561128257606060046101403760506004356004016101a037603060043560040135111561076057600080fd5b604060243560040161022037602060243560040135111561078057600080fd5b60806044356004016102803760606044356004013511156107a057600080fd5b63ffffffff600354106107b257600080fd5b633b9aca0061034052610340516107c857600080fd5b610340513404610320526000546103205110156107e457600080fd5b60306101a051146107f457600080fd5b6020610220511461080457600080fd5b6060610280511461081457600080fd5b6101a0516101c0516101e05161020051610220516102405161026051610280516102a0516102c0516102e05161030051610320516103405161036051610380516103a05163806732896103c052610320516103e0526103e051600658016101ab565b506104405260006104a0525b6104405160206001820306601f82010390506104a0511015156108a4576108bd565b6104a05161046001526104a0516020016104a052610882565b6103a05261038052610360526103405261032052610300526102e0526102c0526102a05261028052610260526102405261022052610200526101e0526101c0526101a052610440805160200180610360828460006004600a8704601201f161092457600080fd5b50506101a0516101c0516101e05161020051610220516102405161026051610280516102a0516102c0516102e05161030051610320516103405161036051610380516103a0516103c0516103e05161040051610420516104405161046051610480516104a05163806732896104c0526003546104e0526104e051600658016101ab565b506105405260006105a0525b6105405160206001820306601f82010390506105a0511015156109d5576109ee565b6105a05161056001526105a0516020016105a0526109b3565b6104a05261048052610460526104405261042052610400526103e0526103c0526103a05261038052610360526103405261032052610300526102e0526102c0526102a05261028052610260526102405261022052610200526101e0526101c0526101a0526105408051602001806105c0828460006004600a8704601201f1610a7557600080fd5b505060a06106405261064051610680526101a08051602001806106405161068001828460006004600a8704601201f1610aad57600080fd5b505061064051610680015160206001820306601f8201039050610640516106800161062081516040818352015b8361062051101515610aeb57610b08565b6000610620516020850101535b8151600101808352811415610ada575b50505050602061064051610680015160206001820306601f820103905061064051010161064052610640516106a0526102208051602001806106405161068001828460006004600a8704601201f1610b5f57600080fd5b505061064051610680015160206001820306601f8201039050610640516106800161062081516020818352015b8361062051101515610b9d57610bba565b6000610620516020850101535b8151600101808352811415610b8c575b50505050602061064051610680015160206001820306601f820103905061064051010161064052610640516106c0526103608051602001806106405161068001828460006004600a8704601201f1610c1157600080fd5b505061064051610680015160206001820306601f8201039050610640516106800161062081516020818352015b8361062051101515610c4f57610c6c565b6000610620516020850101535b8151600101808352811415610c3e575b50505050602061064051610680015160206001820306601f820103905061064051010161064052610640516106e0526102808051602001806106405161068001828460006004600a8704601201f1610cc357600080fd5b505061064051610680015160206001820306601f8201039050610640516106800161062081516060818352015b8361062051101515610d0157610d1e565b6000610620516020850101535b8151600101808352811415610cf0575b50505050602061064051610680015160206001820306601f82010390506106405101016106405261064051610700526105c08051602001806106405161068001828460006004600a8704601201f1610d7557600080fd5b505061064051610680015160206001820306601f8201039050610640516106800161062081516020818352015b8361062051101515610db357610dd0565b6000610620516020850101535b8151600101808352811415610da2575b50505050602061064051610680015160206001820306601f8201039050610640510101610640527f649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c561064051610680a160006107205260006101a06030806020846107e001018260208501600060046016f150508051820191505060006010602082066107600160208284011115610e6757600080fd5b60208061078082610720600060046015f15050818152809050905090506010806020846107e001018260208501600060046013f1505080518201915050806107e0526107e09050602060c0825160208401600060025af1610ec757600080fd5b60c0519050610740526000600060406020820661088001610280518284011115610ef057600080fd5b6060806108a0826020602088068803016102800160006004601bf1505081815280905090509050602060c0825160208401600060025af1610f3057600080fd5b60c0519050602082610a800101526020810190506000604060206020820661094001610280518284011115610f6457600080fd5b606080610960826020602088068803016102800160006004601bf1505081815280905090509050602080602084610a0001018260208501600060046015f150508051820191505061072051602082610a0001015260208101905080610a0052610a009050602060c0825160208401600060025af1610fe157600080fd5b60c0519050602082610a8001015260208101905080610a8052610a809050602060c0825160208401600060025af161101857600080fd5b60c0519050610860526000600061074051602082610b20010152602081019050610220602080602084610b2001018260208501600060046015f150508051820191505080610b2052610b209050602060c0825160208401600060025af161107e57600080fd5b60c0519050602082610ca00101526020810190506000610360600880602084610c2001018260208501600060046012f15050805182019150506000601860208206610ba001602082840111156110d357600080fd5b602080610bc082610720600060046015f1505081815280905090509050601880602084610c2001018260208501600060046014f150508051820191505061086051602082610c2001015260208101905080610c2052610c209050602060c0825160208401600060025af161114657600080fd5b60c0519050602082610ca001015260208101905080610ca052610ca09050602060c0825160208401600060025af161117d57600080fd5b60c0519050610b0052600380546001825401101561119a57600080fd5b6001815401815550600354610d2052610d4060006020818352015b60016001610d20511614156111ea57610b0051610d4051602081106111d957600080fd5b600260c052602060c020015561127e565b6000610d4051602081106111fd57600080fd5b600260c052602060c0200154602082610d60010152602081019050610b0051602082610d6001015260208101905080610d6052610d609050602060c0825160208401600060025af161124e57600080fd5b60c0519050610b0052610d20600261126557600080fd5b60028151048152505b81516001018083528114156111b5575b5050005b639890220b60005114156112b657341561129b57600080fd5b600060006000600030316004546000f16112b457600080fd5b005b631ea30fef60005114156112dc5734156112cf57600080fd5b60005460005260206000f3005b63eb8545ee60005114156113025734156112f557600080fd5b60035460005260206000f3005b638ba35cdf600051141561132857341561131b57600080fd5b60045460005260206000f3005b60006000fd5b6101b96114e7036101b96000396101b96114e7036000f3`

// DeployDepositContract deploys a new Ethereum contract, binding an instance of DepositContract to it.
func DeployDepositContract(auth *bind.TransactOpts, backend bind.ContractBackend, minDeposit *big.Int, _drain_address common.Address) (common.Address, *types.Transaction, *DepositContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DepositContractBin), backend, minDeposit, _drain_address)
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

// MINDEPOSITAMOUNT is a free data retrieval call binding the contract method 0x1ea30fef.
//
// Solidity: function MIN_DEPOSIT_AMOUNT() constant returns(uint256 out)
func (_DepositContract *DepositContractCaller) MINDEPOSITAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "MIN_DEPOSIT_AMOUNT")
	return *ret0, err
}

// MINDEPOSITAMOUNT is a free data retrieval call binding the contract method 0x1ea30fef.
//
// Solidity: function MIN_DEPOSIT_AMOUNT() constant returns(uint256 out)
func (_DepositContract *DepositContractSession) MINDEPOSITAMOUNT() (*big.Int, error) {
	return _DepositContract.Contract.MINDEPOSITAMOUNT(&_DepositContract.CallOpts)
}

// MINDEPOSITAMOUNT is a free data retrieval call binding the contract method 0x1ea30fef.
//
// Solidity: function MIN_DEPOSIT_AMOUNT() constant returns(uint256 out)
func (_DepositContract *DepositContractCallerSession) MINDEPOSITAMOUNT() (*big.Int, error) {
	return _DepositContract.Contract.MINDEPOSITAMOUNT(&_DepositContract.CallOpts)
}

// DepositCount is a free data retrieval call binding the contract method 0xeb8545ee.
//
// Solidity: function deposit_count() constant returns(uint256 out)
func (_DepositContract *DepositContractCaller) DepositCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "deposit_count")
	return *ret0, err
}

// DepositCount is a free data retrieval call binding the contract method 0xeb8545ee.
//
// Solidity: function deposit_count() constant returns(uint256 out)
func (_DepositContract *DepositContractSession) DepositCount() (*big.Int, error) {
	return _DepositContract.Contract.DepositCount(&_DepositContract.CallOpts)
}

// DepositCount is a free data retrieval call binding the contract method 0xeb8545ee.
//
// Solidity: function deposit_count() constant returns(uint256 out)
func (_DepositContract *DepositContractCallerSession) DepositCount() (*big.Int, error) {
	return _DepositContract.Contract.DepositCount(&_DepositContract.CallOpts)
}

// DrainAddress is a free data retrieval call binding the contract method 0x8ba35cdf.
//
// Solidity: function drain_address() constant returns(address out)
func (_DepositContract *DepositContractCaller) DrainAddress(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "drain_address")
	return *ret0, err
}

// DrainAddress is a free data retrieval call binding the contract method 0x8ba35cdf.
//
// Solidity: function drain_address() constant returns(address out)
func (_DepositContract *DepositContractSession) DrainAddress() (common.Address, error) {
	return _DepositContract.Contract.DrainAddress(&_DepositContract.CallOpts)
}

// DrainAddress is a free data retrieval call binding the contract method 0x8ba35cdf.
//
// Solidity: function drain_address() constant returns(address out)
func (_DepositContract *DepositContractCallerSession) DrainAddress() (common.Address, error) {
	return _DepositContract.Contract.DrainAddress(&_DepositContract.CallOpts)
}

// GetDepositCount is a free data retrieval call binding the contract method 0x621fd130.
//
// Solidity: function get_deposit_count() constant returns(bytes out)
func (_DepositContract *DepositContractCaller) GetDepositCount(opts *bind.CallOpts) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "get_deposit_count")
	return *ret0, err
}

// GetDepositCount is a free data retrieval call binding the contract method 0x621fd130.
//
// Solidity: function get_deposit_count() constant returns(bytes out)
func (_DepositContract *DepositContractSession) GetDepositCount() ([]byte, error) {
	return _DepositContract.Contract.GetDepositCount(&_DepositContract.CallOpts)
}

// GetDepositCount is a free data retrieval call binding the contract method 0x621fd130.
//
// Solidity: function get_deposit_count() constant returns(bytes out)
func (_DepositContract *DepositContractCallerSession) GetDepositCount() ([]byte, error) {
	return _DepositContract.Contract.GetDepositCount(&_DepositContract.CallOpts)
}

// GetHashTreeRoot is a free data retrieval call binding the contract method 0x863a311b.
//
// Solidity: function get_hash_tree_root() constant returns(bytes32 out)
func (_DepositContract *DepositContractCaller) GetHashTreeRoot(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "get_hash_tree_root")
	return *ret0, err
}

// GetHashTreeRoot is a free data retrieval call binding the contract method 0x863a311b.
//
// Solidity: function get_hash_tree_root() constant returns(bytes32 out)
func (_DepositContract *DepositContractSession) GetHashTreeRoot() ([32]byte, error) {
	return _DepositContract.Contract.GetHashTreeRoot(&_DepositContract.CallOpts)
}

// GetHashTreeRoot is a free data retrieval call binding the contract method 0x863a311b.
//
// Solidity: function get_hash_tree_root() constant returns(bytes32 out)
func (_DepositContract *DepositContractCallerSession) GetHashTreeRoot() ([32]byte, error) {
	return _DepositContract.Contract.GetHashTreeRoot(&_DepositContract.CallOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xc47e300d.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature) returns()
func (_DepositContract *DepositContractTransactor) Deposit(opts *bind.TransactOpts, pubkey []byte, withdrawal_credentials []byte, signature []byte) (*types.Transaction, error) {
	return _DepositContract.contract.Transact(opts, "deposit", pubkey, withdrawal_credentials, signature)
}

// Deposit is a paid mutator transaction binding the contract method 0xc47e300d.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature) returns()
func (_DepositContract *DepositContractSession) Deposit(pubkey []byte, withdrawal_credentials []byte, signature []byte) (*types.Transaction, error) {
	return _DepositContract.Contract.Deposit(&_DepositContract.TransactOpts, pubkey, withdrawal_credentials, signature)
}

// Deposit is a paid mutator transaction binding the contract method 0xc47e300d.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature) returns()
func (_DepositContract *DepositContractTransactorSession) Deposit(pubkey []byte, withdrawal_credentials []byte, signature []byte) (*types.Transaction, error) {
	return _DepositContract.Contract.Deposit(&_DepositContract.TransactOpts, pubkey, withdrawal_credentials, signature)
}

// Drain is a paid mutator transaction binding the contract method 0x9890220b.
//
// Solidity: function drain() returns()
func (_DepositContract *DepositContractTransactor) Drain(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DepositContract.contract.Transact(opts, "drain")
}

// Drain is a paid mutator transaction binding the contract method 0x9890220b.
//
// Solidity: function drain() returns()
func (_DepositContract *DepositContractSession) Drain() (*types.Transaction, error) {
	return _DepositContract.Contract.Drain(&_DepositContract.TransactOpts)
}

// Drain is a paid mutator transaction binding the contract method 0x9890220b.
//
// Solidity: function drain() returns()
func (_DepositContract *DepositContractTransactorSession) Drain() (*types.Transaction, error) {
	return _DepositContract.Contract.Drain(&_DepositContract.TransactOpts)
}

// DepositContractDepositEventIterator is returned from FilterDepositEvent and is used to iterate over the raw logs and unpacked data for DepositEvent events raised by the DepositContract contract.
type DepositContractDepositEventIterator struct {
	Event *DepositContractDepositEvent // Event containing the contract specifics and raw log

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
func (it *DepositContractDepositEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DepositContractDepositEvent)
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
		it.Event = new(DepositContractDepositEvent)
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
func (it *DepositContractDepositEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DepositContractDepositEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DepositContractDepositEvent represents a DepositEvent event raised by the DepositContract contract.
type DepositContractDepositEvent struct {
	Pubkey                []byte
	WithdrawalCredentials []byte
	Amount                []byte
	Signature             []byte
	Index                 []byte
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterDepositEvent is a free log retrieval operation binding the contract event 0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5.
//
// Solidity: event DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
func (_DepositContract *DepositContractFilterer) FilterDepositEvent(opts *bind.FilterOpts) (*DepositContractDepositEventIterator, error) {

	logs, sub, err := _DepositContract.contract.FilterLogs(opts, "DepositEvent")
	if err != nil {
		return nil, err
	}
	return &DepositContractDepositEventIterator{contract: _DepositContract.contract, event: "DepositEvent", logs: logs, sub: sub}, nil
}

// WatchDepositEvent is a free log subscription operation binding the contract event 0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5.
//
// Solidity: event DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
func (_DepositContract *DepositContractFilterer) WatchDepositEvent(opts *bind.WatchOpts, sink chan<- *DepositContractDepositEvent) (event.Subscription, error) {

	logs, sub, err := _DepositContract.contract.WatchLogs(opts, "DepositEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DepositContractDepositEvent)
				if err := _DepositContract.contract.UnpackLog(event, "DepositEvent", log); err != nil {
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
