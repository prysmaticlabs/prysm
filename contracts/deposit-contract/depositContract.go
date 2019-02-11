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
const DepositContractABI = "[{\"name\":\"Deposit\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"data\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"merkle_tree_index\",\"indexed\":false},{\"type\":\"bytes32[32]\",\"name\":\"branch\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"ChainStart\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"time\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"__init__\",\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"depositThreshold\"},{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"uint256\",\"name\":\"maxDeposit\"},{\"type\":\"bool\",\"name\":\"skipChainstartDelay\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"to_little_endian_64\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"value\"}],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":15330},{\"name\":\"get_deposit_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":30835},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"deposit_input\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":478343},{\"name\":\"chainStarted\",\"outputs\":[{\"type\":\"bool\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":603}]"

// DepositContractBin is the compiled bytecode used for deploying new contracts.
const DepositContractBin = `0x600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a052608061191f6101403934156100a757600080fd5b6020606061191f0160c03960c051600281106100c257600080fd5b506101405160005561016051600155610180516002556101a0516007556101c06000601f818352015b60006101c051602081106100fe57600080fd5b600360c052602060c02001546020826101e00101526020810190506101c0516020811061012a57600080fd5b600360c052602060c02001546020826101e0010152602081019050806101e0526101e0905080516020820120905060605160016101c051018060405190131561017257600080fd5b809190121561018057600080fd5b6020811061018d57600080fd5b600360c052602060c020015560605160016101c05101806040519013156101b357600080fd5b80919012156101c157600080fd5b602081106101ce57600080fd5b600360c052602060c020015460605160016101c05101806040519013156101f457600080fd5b809190121561020257600080fd5b6020811061020f57600080fd5b600460c052602060c02001555b81516001018083528114156100eb575b505061190756600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a0526000156101a3575b6101605261014052601860086020820661020001602082840111156100bf57600080fd5b60208061022082610140600060046015f15050818152809050905090508051602001806102c0828460006004600a8704601201f16100fc57600080fd5b50506102c05160206001820306601f82010390506103206102c0516008818352015b8261032051111561012e5761014a565b6000610320516102e001535b815160010180835281141561011e575b50505060206102a05260406102c0510160206001820306601f8201039050610280525b60006102805111151561017f5761019b565b602061028051036102a00151602061028051036102805261016d565b610160515650005b638067328960005114156104f957602060046101403734156101c457600080fd5b67ffffffffffffffff6101405111156101dc57600080fd5b6101405161016051610180516101a05163b0429c706101c052610140516101e0526101e0516006580161009b565b506102405260006102a0525b6102405160206001820306601f82010390506102a05110151561023857610251565b6102a05161026001526102a0516020016102a052610216565b6101a052610180526101605261014052610240805160200180610160828460006004600a8704601201f161028457600080fd5b50506101608060200151600082518060209013156102a157600080fd5b80919012156102af57600080fd5b806020036101000a82049050905090506102c05260006102e05261030060006008818352015b6102e051600860008112156102f2578060000360020a82046102f9565b8060020a82025b905090506102e05260ff6102c05116610320526102e051610320516102e05101101561032457600080fd5b610320516102e051016102e0526102c0517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8600081121561036d578060000360020a8204610374565b8060020a82025b905090506102c0525b81516001018083528114156102d5575b50506101405161016051610180516101a0516101c0516101e05161020051610220516102405161026051610280516102a0516102c0516102e05163b0429c70610340526102e05161036052610360516006580161009b565b506103c0526000610420525b6103c05160206001820306601f8201039050610420511015156104135761042c565b610420516103e0015261042051602001610420526103f1565b6102e0526102c0526102a05261028052610260526102405261022052610200526101e0526101c0526101a0526101805261016052610140526103c0805160200180610480828460006004600a8704601201f161048757600080fd5b50506104805160206001820306601f82010390506104e0610480516008818352015b826104e05111156104b9576104d5565b60006104e0516104a001535b81516001018083528114156104a9575b5050506020610460526040610480510160206001820306601f8201039050610460f3005b63c5f2892f600051141561063257341561051257600080fd5b6000610140526005546101605261018060006020818352015b6001600261053857600080fd5b6002610160510614156105a2576000610180516020811061055857600080fd5b600460c052602060c02001546020826102200101526020810190506101405160208261022001015260208101905080610220526102209050805160208201209050610140526105fb565b6000610140516020826101a001015260208101905061018051602081106105c857600080fd5b600360c052602060c02001546020826101a0010152602081019050806101a0526101a09050805160208201209050610140525b610160600261060957600080fd5b60028151048152505b815160010180835281141561052b575b50506101405160005260206000f3005b6398b1e06a60005114156116a85760206004610140376102206004356004016101603761020060043560040135111561066a57600080fd5b633b9aca006103c0526103c05161068057600080fd5b6103c05134046103a0526001546103a051101561069c57600080fd5b6002546103a05111156106ae57600080fd5b6005546103e05242610400526000606061070060246380673289610680526103a0516106a05261069c6000305af16106e557600080fd5b61072060088060208461084001018260208501600060046012f150508051820191505060606107e06024638067328961076052610400516107805261077c6000305af161073157600080fd5b61080060088060208461084001018260208501600060046012f15050805182019150506101606102008060208461084001018260208501600060046045f150508051820191505080610840526108409050805160200180610420828460006004600a8704601201f16107a257600080fd5b50506000610aa0526002610ac052610ae060006020818352015b6000610ac0516107cb57600080fd5b610ac0516103e05160016103e0510110156107e557600080fd5b60016103e05101061415156107f957610865565b610aa060605160018251018060405190131561081457600080fd5b809190121561082257600080fd5b815250610ac080511515610837576000610851565b600281516002835102041461084b57600080fd5b60028151025b8152505b81516001018083528114156107bc575b5050610420805160208201209050610b0052610b2060006020818352015b610aa051610b205112156108ea576000610b2051602081106108a457600080fd5b600460c052602060c0200154602082610b40010152602081019050610b0051602082610b4001015260208101905080610b4052610b409050805160208201209050610b00525b5b8151600101808352811415610883575b5050610b0051610aa0516020811061091257600080fd5b600460c052602060c0200155600580546001825401101561093257600080fd5b60018154018155506020610c40600463c5f2892f610be052610bfc6000305af161095b57600080fd5b610c4051610bc0526060610ce060246380673289610c60526103e051610c8052610c7c6000305af161098c57600080fd5b610d00805160200180610d40828460006004600a8704601201f16109af57600080fd5b5050610bc051610e0052600460c052602060c02054610e60526001600460c052602060c0200154610e80526002600460c052602060c0200154610ea0526003600460c052602060c0200154610ec0526004600460c052602060c0200154610ee0526005600460c052602060c0200154610f00526006600460c052602060c0200154610f20526007600460c052602060c0200154610f40526008600460c052602060c0200154610f60526009600460c052602060c0200154610f8052600a600460c052602060c0200154610fa052600b600460c052602060c0200154610fc052600c600460c052602060c0200154610fe052600d600460c052602060c020015461100052600e600460c052602060c020015461102052600f600460c052602060c0200154611040526010600460c052602060c0200154611060526011600460c052602060c0200154611080526012600460c052602060c02001546110a0526013600460c052602060c02001546110c0526014600460c052602060c02001546110e0526015600460c052602060c0200154611100526016600460c052602060c0200154611120526017600460c052602060c0200154611140526018600460c052602060c0200154611160526019600460c052602060c020015461118052601a600460c052602060c02001546111a052601b600460c052602060c02001546111c052601c600460c052602060c02001546111e052601d600460c052602060c020015461120052601e600460c052602060c020015461122052601f600460c052602060c020015461124052610460610dc052610dc051610e2052610420805160200180610dc051610e0001828460006004600a8704601201f1610c2d57600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da08151610220818352015b83610da051101515610c6c57610c89565b6000610da0516020850101535b8151600101808352811415610c5b575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc052610dc051610e4052610d40805160200180610dc051610e0001828460006004600a8704601201f1610ce057600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516020818352015b83610da051101515610d1e57610d3b565b6000610da0516020850101535b8151600101808352811415610d0d575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc0527fce7a77a358682d6c81f71216fb7fb108b03bc8badbf67f5d131ba5363cbefb42610dc051610e00a16002546103a05114156116a6576006805460018254011015610dac57600080fd5b600181540181555060005460065414156116a55760075415611461576020611500600463c5f2892f6114a0526114bc6000305af1610de957600080fd5b611500516115205261016051610180516101a0516101c0516101e05161020051610220516102405161026051610280516102a0516102c0516102e05161030051610320516103405161036051610380516103a0516103c0516103e05161040051610420516104405161046051610480516104a0516104c0516104e05161050051610520516105405161056051610580516105a0516105c0516105e05161060051610620516106405161066051610680516106a0516106c0516106e05161070051610720516107405161076051610780516107a0516107c0516107e05161080051610820516108405161086051610880516108a0516108c0516108e05161090051610920516109405161096051610980516109a0516109c0516109e051610a0051610a2051610a4051610a6051610a8051610aa051610ac051610ae051610b0051610b2051610b4051610b6051610b8051610ba051610bc051610be051610c0051610c2051610c4051610c6051610c8051610ca051610cc051610ce051610d0051610d2051610d4051610d6051610d8051610da051610dc051610de051610e0051610e2051610e4051610e6051610e8051610ea051610ec051610ee051610f0051610f2051610f4051610f6051610f8051610fa051610fc051610fe05161100051611020516110405161106051611080516110a0516110c0516110e05161110051611120516111405161116051611180516111a0516111c0516111e05161120051611220516112405161126051611280516112a0516112c0516112e05161130051611320516113405161136051611380516113a0516113c0516113e05161140051611420516114405161146051611480516114a0516114c0516114e051611500516115205163b0429c70611540526104005161156052611560516006580161009b565b506115c0526000611620525b6115c05160206001820306601f8201039050611620511015156110b9576110d2565b611620516115e001526116205160200161162052611097565b61152052611500526114e0526114c0526114a05261148052611460526114405261142052611400526113e0526113c0526113a05261138052611360526113405261132052611300526112e0526112c0526112a05261128052611260526112405261122052611200526111e0526111c0526111a05261118052611160526111405261112052611100526110e0526110c0526110a0526110805261106052611040526110205261100052610fe052610fc052610fa052610f8052610f6052610f4052610f2052610f0052610ee052610ec052610ea052610e8052610e6052610e4052610e2052610e0052610de052610dc052610da052610d8052610d6052610d4052610d2052610d0052610ce052610cc052610ca052610c8052610c6052610c4052610c2052610c0052610be052610bc052610ba052610b8052610b6052610b4052610b2052610b0052610ae052610ac052610aa052610a8052610a6052610a4052610a2052610a00526109e0526109c0526109a05261098052610960526109405261092052610900526108e0526108c0526108a05261088052610860526108405261082052610800526107e0526107c0526107a05261078052610760526107405261072052610700526106e0526106c0526106a05261068052610660526106405261062052610600526105e0526105c0526105a05261058052610560526105405261052052610500526104e0526104c0526104a05261048052610460526104405261042052610400526103e0526103c0526103a05261038052610360526103405261032052610300526102e0526102c0526102a05261028052610260526102405261022052610200526101e0526101c0526101a05261018052610160526115c0805160200180611640828460006004600a8704601201f161137157600080fd5b5050611520516117005260406116c0526116c051611720526116408051602001806116c05161170001828460006004600a8704601201f16113b157600080fd5b50506116c051611700015160206001820306601f82010390506116c051611700016116a081516020818352015b836116a0511015156113ef5761140c565b60006116a0516020850101535b81516001018083528114156113de575b5050505060206116c051611700015160206001820306601f82010390506116c05101016116c0527fd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc6116c051611700a161169f565b4261128052426112a0526201518061147857600080fd5b620151806112a0510661128051101561149057600080fd5b426112a052620151806114a257600080fd5b620151806112a051066112805103620151804261128052426112a052620151806114cb57600080fd5b620151806112a051066112805110156114e357600080fd5b426112a052620151806114f557600080fd5b620151806112a05106611280510301101561150f57600080fd5b620151804261128052426112a0526201518061152a57600080fd5b620151806112a0510661128051101561154257600080fd5b426112a0526201518061155457600080fd5b620151806112a05106611280510301611260526060611340602463806732896112c052611260516112e0526112dc6000305af161159057600080fd5b6113608051602001806113a0828460006004600a8704601201f16115b357600080fd5b5050610bc0516114605260406114205261142051611480526113a08051602001806114205161146001828460006004600a8704601201f16115f357600080fd5b505061142051611460015160206001820306601f8201039050611420516114600161140081516020818352015b83611400511015156116315761164e565b6000611400516020850101535b8151600101808352811415611620575b50505050602061142051611460015160206001820306601f8201039050611420510101611420527fd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc61142051611460a15b60016008555b5b005b63845980e860005114156116ce5734156116c157600080fd5b60085460005260206000f3005b60006000fd5b61023361190703610233600039610233611907036000f3`

// DeployDepositContract deploys a new Ethereum contract, binding an instance of DepositContract to it.
func DeployDepositContract(auth *bind.TransactOpts, backend bind.ContractBackend, depositThreshold *big.Int, minDeposit *big.Int, maxDeposit *big.Int, skipChainstartDelay bool) (common.Address, *types.Transaction, *DepositContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DepositContractBin), backend, depositThreshold, minDeposit, maxDeposit, skipChainstartDelay)
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
