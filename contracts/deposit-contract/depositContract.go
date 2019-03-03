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
const DepositContractABI = "[{\"name\":\"Deposit\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"data\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"merkle_tree_index\",\"indexed\":false},{\"type\":\"bytes32[32]\",\"name\":\"branch\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"ChainStart\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"time\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"__init__\",\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"depositThreshold\"},{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"uint256\",\"name\":\"maxDeposit\"},{\"type\":\"uint256\",\"name\":\"customChainstartDelay\"},{\"type\":\"address\",\"name\":\"_drain_address\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"to_little_endian_64\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"value\"}],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":15330},{\"name\":\"get_deposit_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":30835},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"deposit_input\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":535502},{\"name\":\"drain\",\"outputs\":[],\"inputs\":[],\"constant\":false,\"payable\":false,\"type\":\"function\",\"gas\":35793},{\"name\":\"CHAIN_START_FULL_DEPOSIT_THRESHOLD\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":633},{\"name\":\"MIN_DEPOSIT_AMOUNT\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":663},{\"name\":\"MAX_DEPOSIT_AMOUNT\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":693},{\"name\":\"deposit_count\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":723},{\"name\":\"full_deposit_count\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":753},{\"name\":\"custom_chainstart_delay\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":783},{\"name\":\"genesisTime\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":3036},{\"name\":\"drain_address\",\"outputs\":[{\"type\":\"address\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":843}]"

// DepositContractBin is the compiled bytecode used for deploying new contracts.
const DepositContractBin = `0x600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a05260a06117b86101403934156100a757600080fd5b602060806117b80160c03960c05160205181106100c357600080fd5b506101405160005561016051600155610180516002556101a0516007556101c0516009556101e06000601f818352015b60006101e0516020811061010657600080fd5b600360c052602060c02001546020826102000101526020810190506101e0516020811061013257600080fd5b600360c052602060c02001546020826102000101526020810190508061020052610200905080516020820120905060605160016101e051018060405190131561017a57600080fd5b809190121561018857600080fd5b6020811061019557600080fd5b600360c052602060c020015560605160016101e05101806040519013156101bb57600080fd5b80919012156101c957600080fd5b602081106101d657600080fd5b600360c052602060c020015460605160016101e05101806040519013156101fc57600080fd5b809190121561020a57600080fd5b6020811061021757600080fd5b600460c052602060c02001555b81516001018083528114156100f3575b50506117a056600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a0526000156101a3575b6101605261014052601860086020820661020001602082840111156100bf57600080fd5b60208061022082610140600060046015f15050818152809050905090508051602001806102c0828460006004600a8704601201f16100fc57600080fd5b50506102c05160206001820306601f82010390506103206102c0516008818352015b8261032051111561012e5761014a565b6000610320516102e001535b815160010180835281141561011e575b50505060206102a05260406102c0510160206001820306601f8201039050610280525b60006102805111151561017f5761019b565b602061028051036102a00151602061028051036102805261016d565b610160515650005b638067328960005114156104f957602060046101403734156101c457600080fd5b67ffffffffffffffff6101405111156101dc57600080fd5b6101405161016051610180516101a05163b0429c706101c052610140516101e0526101e0516006580161009b565b506102405260006102a0525b6102405160206001820306601f82010390506102a05110151561023857610251565b6102a05161026001526102a0516020016102a052610216565b6101a052610180526101605261014052610240805160200180610160828460006004600a8704601201f161028457600080fd5b50506101608060200151600082518060209013156102a157600080fd5b80919012156102af57600080fd5b806020036101000a82049050905090506102c05260006102e05261030060006008818352015b6102e051600860008112156102f2578060000360020a82046102f9565b8060020a82025b905090506102e05260ff6102c05116610320526102e051610320516102e05101101561032457600080fd5b610320516102e051016102e0526102c0517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8600081121561036d578060000360020a8204610374565b8060020a82025b905090506102c0525b81516001018083528114156102d5575b50506101405161016051610180516101a0516101c0516101e05161020051610220516102405161026051610280516102a0516102c0516102e05163b0429c70610340526102e05161036052610360516006580161009b565b506103c0526000610420525b6103c05160206001820306601f8201039050610420511015156104135761042c565b610420516103e0015261042051602001610420526103f1565b6102e0526102c0526102a05261028052610260526102405261022052610200526101e0526101c0526101a0526101805261016052610140526103c0805160200180610480828460006004600a8704601201f161048757600080fd5b50506104805160206001820306601f82010390506104e0610480516008818352015b826104e05111156104b9576104d5565b60006104e0516104a001535b81516001018083528114156104a9575b5050506020610460526040610480510160206001820306601f8201039050610460f3005b63c5f2892f600051141561063257341561051257600080fd5b6000610140526005546101605261018060006020818352015b6001600261053857600080fd5b6002610160510614156105a2576000610180516020811061055857600080fd5b600460c052602060c02001546020826102200101526020810190506101405160208261022001015260208101905080610220526102209050805160208201209050610140526105fb565b6000610140516020826101a001015260208101905061018051602081106105c857600080fd5b600360c052602060c02001546020826101a0010152602081019050806101a0526101a09050805160208201209050610140525b610160600261060957600080fd5b60028151048152505b815160010180835281141561052b575b50506101405160005260206000f3005b6398b1e06a600051141561133e5760206004610140376102206004356004016101603761020060043560040135111561066a57600080fd5b633b9aca006103c0526103c05161068057600080fd5b6103c05134046103a0526001546103a051101561069c57600080fd5b6002546103a05111156106ae57600080fd5b6005546103e05242610400526000606061070060246380673289610680526103a0516106a05261069c6000305af16106e557600080fd5b61072060088060208461084001018260208501600060046012f150508051820191505060606107e06024638067328961076052610400516107805261077c6000305af161073157600080fd5b61080060088060208461084001018260208501600060046012f15050805182019150506101606102008060208461084001018260208501600060046045f150508051820191505080610840526108409050805160200180610420828460006004600a8704601201f16107a257600080fd5b50506000610aa0526002610ac052610ae060006020818352015b6000610ac0516107cb57600080fd5b610ac0516103e05160016103e0510110156107e557600080fd5b60016103e05101061415156107f957610865565b610aa060605160018251018060405190131561081457600080fd5b809190121561082257600080fd5b815250610ac080511515610837576000610851565b600281516002835102041461084b57600080fd5b60028151025b8152505b81516001018083528114156107bc575b5050610420805160208201209050610b0052610b2060006020818352015b610aa051610b205112156108ea576000610b2051602081106108a457600080fd5b600460c052602060c0200154602082610b40010152602081019050610b0051602082610b4001015260208101905080610b4052610b409050805160208201209050610b00525b5b8151600101808352811415610883575b5050610b0051610aa0516020811061091257600080fd5b600460c052602060c0200155600580546001825401101561093257600080fd5b60018154018155506020610c40600463c5f2892f610be052610bfc6000305af161095b57600080fd5b610c4051610bc0526060610ce060246380673289610c60526103e051610c8052610c7c6000305af161098c57600080fd5b610d00805160200180610d40828460006004600a8704601201f16109af57600080fd5b5050610bc051610e0052600460c052602060c02054610e60526001600460c052602060c0200154610e80526002600460c052602060c0200154610ea0526003600460c052602060c0200154610ec0526004600460c052602060c0200154610ee0526005600460c052602060c0200154610f00526006600460c052602060c0200154610f20526007600460c052602060c0200154610f40526008600460c052602060c0200154610f60526009600460c052602060c0200154610f8052600a600460c052602060c0200154610fa052600b600460c052602060c0200154610fc052600c600460c052602060c0200154610fe052600d600460c052602060c020015461100052600e600460c052602060c020015461102052600f600460c052602060c0200154611040526010600460c052602060c0200154611060526011600460c052602060c0200154611080526012600460c052602060c02001546110a0526013600460c052602060c02001546110c0526014600460c052602060c02001546110e0526015600460c052602060c0200154611100526016600460c052602060c0200154611120526017600460c052602060c0200154611140526018600460c052602060c0200154611160526019600460c052602060c020015461118052601a600460c052602060c02001546111a052601b600460c052602060c02001546111c052601c600460c052602060c02001546111e052601d600460c052602060c020015461120052601e600460c052602060c020015461122052601f600460c052602060c020015461124052610460610dc052610dc051610e2052610420805160200180610dc051610e0001828460006004600a8704601201f1610c2d57600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da08151610220818352015b83610da051101515610c6c57610c89565b6000610da0516020850101535b8151600101808352811415610c5b575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc052610dc051610e4052610d40805160200180610dc051610e0001828460006004600a8704601201f1610ce057600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516020818352015b83610da051101515610d1e57610d3b565b6000610da0516020850101535b8151600101808352811415610d0d575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc0527fce7a77a358682d6c81f71216fb7fb108b03bc8badbf67f5d131ba5363cbefb42610dc051610e00a16002546103a051141561133c576006805460018254011015610dac57600080fd5b6001815401815550600054600654141561133b57600060075411156110925742611460524261148052600754610de157600080fd5b6007546114805106611460511015610df857600080fd5b4261148052600754610e0957600080fd5b6007546114805106611460510360075442611460524261148052600754610e2f57600080fd5b6007546114805106611460511015610e4657600080fd5b4261148052600754610e5757600080fd5b60075461148051066114605103011015610e7057600080fd5b60075442611460524261148052600754610e8957600080fd5b6007546114805106611460511015610ea057600080fd5b4261148052600754610eb157600080fd5b6007546114805106611460510301611440526060611520602463806732896114a052611440516114c0526114bc6000305af1610eec57600080fd5b61154080600860c052602060c020602082510161012060006002818352015b82610120516020021115610f1e57610f40565b61012051602002850151610120518501555b8151600101808352811415610f0b575b50505050505060206115e0600463c5f2892f6115805261159c6000305af1610f6757600080fd5b6115e051611600526116005161168052604061164052611640516116a05260088060c052602060c0206116405161168001602082540161012060006002818352015b82610120516020021115610fbc57610fde565b61012051850154610120516020028501525b8151600101808352811415610fa9575b50505050505061164051611680015160206001820306601f8201039050611640516116800161162081516020818352015b83611620511015156110205761103d565b6000611620516020850101535b815160010180835281141561100f575b50505050602061164051611680015160206001820306601f8201039050611640510101611640527fd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc61164051611680a161133a565b4261128052426112a052620151806110a957600080fd5b620151806112a051066112805110156110c157600080fd5b426112a052620151806110d357600080fd5b620151806112a051066112805103620151804261128052426112a052620151806110fc57600080fd5b620151806112a0510661128051101561111457600080fd5b426112a0526201518061112657600080fd5b620151806112a05106611280510301101561114057600080fd5b620151804261128052426112a0526201518061115b57600080fd5b620151806112a0510661128051101561117357600080fd5b426112a0526201518061118557600080fd5b620151806112a05106611280510301611260526060611340602463806732896112c052611260516112e0526112dc6000305af16111c157600080fd5b61136080600860c052602060c020602082510161012060006002818352015b826101205160200211156111f357611215565b61012051602002850151610120518501555b81516001018083528114156111e0575b505050505050610bc0516114005260406113c0526113c0516114205260088060c052602060c0206113c05161140001602082540161012060006002818352015b826101205160200211156112685761128a565b61012051850154610120516020028501525b8151600101808352811415611255575b5050505050506113c051611400015160206001820306601f82010390506113c051611400016113a081516020818352015b836113a0511015156112cc576112e9565b60006113a0516020850101535b81516001018083528114156112bb575b5050505060206113c051611400015160206001820306601f82010390506113c05101016113c0527fd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc6113c051611400a15b5b5b005b639890220b600051141561137257341561135757600080fd5b600060006000600030316009546000f161137057600080fd5b005b634a637042600051141561139857341561138b57600080fd5b60005460005260206000f3005b631ea30fef60005114156113be5734156113b157600080fd5b60015460005260206000f3005b634c34a98260005114156113e45734156113d757600080fd5b60025460005260206000f3005b63eb8545ee600051141561140a5734156113fd57600080fd5b60055460005260206000f3005b63188e6c87600051141561143057341561142357600080fd5b60065460005260206000f3005b63b6080cd2600051141561145657341561144957600080fd5b60075460005260206000f3005b6342c6498a600051141561153957341561146f57600080fd5b60088060c052602060c020610180602082540161012060006002818352015b826101205160200211156114a1576114c3565b61012051850154610120516020028501525b815160010180835281141561148e575b5050505050506101805160206001820306601f82010390506101e0610180516008818352015b826101e05111156114f957611515565b60006101e0516101a001535b81516001018083528114156114e9575b5050506020610160526040610180510160206001820306601f8201039050610160f3005b638ba35cdf600051141561155f57341561155257600080fd5b60095460005260206000f3005b60006000fd5b61023b6117a00361023b60003961023b6117a0036000f3`

// DeployDepositContract deploys a new Ethereum contract, binding an instance of DepositContract to it.
func DeployDepositContract(auth *bind.TransactOpts, backend bind.ContractBackend, depositThreshold *big.Int, minDeposit *big.Int, maxDeposit *big.Int, customChainstartDelay *big.Int, _drain_address common.Address) (common.Address, *types.Transaction, *DepositContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DepositContractBin), backend, depositThreshold, minDeposit, maxDeposit, customChainstartDelay, _drain_address)
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

// CHAINSTARTFULLDEPOSITTHRESHOLD is a free data retrieval call binding the contract method 0x4a637042.
//
// Solidity: function CHAIN_START_FULL_DEPOSIT_THRESHOLD() constant returns(uint256 out)
func (_DepositContract *DepositContractCaller) CHAINSTARTFULLDEPOSITTHRESHOLD(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "CHAIN_START_FULL_DEPOSIT_THRESHOLD")
	return *ret0, err
}

// CHAINSTARTFULLDEPOSITTHRESHOLD is a free data retrieval call binding the contract method 0x4a637042.
//
// Solidity: function CHAIN_START_FULL_DEPOSIT_THRESHOLD() constant returns(uint256 out)
func (_DepositContract *DepositContractSession) CHAINSTARTFULLDEPOSITTHRESHOLD() (*big.Int, error) {
	return _DepositContract.Contract.CHAINSTARTFULLDEPOSITTHRESHOLD(&_DepositContract.CallOpts)
}

// CHAINSTARTFULLDEPOSITTHRESHOLD is a free data retrieval call binding the contract method 0x4a637042.
//
// Solidity: function CHAIN_START_FULL_DEPOSIT_THRESHOLD() constant returns(uint256 out)
func (_DepositContract *DepositContractCallerSession) CHAINSTARTFULLDEPOSITTHRESHOLD() (*big.Int, error) {
	return _DepositContract.Contract.CHAINSTARTFULLDEPOSITTHRESHOLD(&_DepositContract.CallOpts)
}

// MAXDEPOSITAMOUNT is a free data retrieval call binding the contract method 0x4c34a982.
//
// Solidity: function MAX_DEPOSIT_AMOUNT() constant returns(uint256 out)
func (_DepositContract *DepositContractCaller) MAXDEPOSITAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "MAX_DEPOSIT_AMOUNT")
	return *ret0, err
}

// MAXDEPOSITAMOUNT is a free data retrieval call binding the contract method 0x4c34a982.
//
// Solidity: function MAX_DEPOSIT_AMOUNT() constant returns(uint256 out)
func (_DepositContract *DepositContractSession) MAXDEPOSITAMOUNT() (*big.Int, error) {
	return _DepositContract.Contract.MAXDEPOSITAMOUNT(&_DepositContract.CallOpts)
}

// MAXDEPOSITAMOUNT is a free data retrieval call binding the contract method 0x4c34a982.
//
// Solidity: function MAX_DEPOSIT_AMOUNT() constant returns(uint256 out)
func (_DepositContract *DepositContractCallerSession) MAXDEPOSITAMOUNT() (*big.Int, error) {
	return _DepositContract.Contract.MAXDEPOSITAMOUNT(&_DepositContract.CallOpts)
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

// CustomChainstartDelay is a free data retrieval call binding the contract method 0xb6080cd2.
//
// Solidity: function custom_chainstart_delay() constant returns(uint256 out)
func (_DepositContract *DepositContractCaller) CustomChainstartDelay(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "custom_chainstart_delay")
	return *ret0, err
}

// CustomChainstartDelay is a free data retrieval call binding the contract method 0xb6080cd2.
//
// Solidity: function custom_chainstart_delay() constant returns(uint256 out)
func (_DepositContract *DepositContractSession) CustomChainstartDelay() (*big.Int, error) {
	return _DepositContract.Contract.CustomChainstartDelay(&_DepositContract.CallOpts)
}

// CustomChainstartDelay is a free data retrieval call binding the contract method 0xb6080cd2.
//
// Solidity: function custom_chainstart_delay() constant returns(uint256 out)
func (_DepositContract *DepositContractCallerSession) CustomChainstartDelay() (*big.Int, error) {
	return _DepositContract.Contract.CustomChainstartDelay(&_DepositContract.CallOpts)
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

// FullDepositCount is a free data retrieval call binding the contract method 0x188e6c87.
//
// Solidity: function full_deposit_count() constant returns(uint256 out)
func (_DepositContract *DepositContractCaller) FullDepositCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "full_deposit_count")
	return *ret0, err
}

// FullDepositCount is a free data retrieval call binding the contract method 0x188e6c87.
//
// Solidity: function full_deposit_count() constant returns(uint256 out)
func (_DepositContract *DepositContractSession) FullDepositCount() (*big.Int, error) {
	return _DepositContract.Contract.FullDepositCount(&_DepositContract.CallOpts)
}

// FullDepositCount is a free data retrieval call binding the contract method 0x188e6c87.
//
// Solidity: function full_deposit_count() constant returns(uint256 out)
func (_DepositContract *DepositContractCallerSession) FullDepositCount() (*big.Int, error) {
	return _DepositContract.Contract.FullDepositCount(&_DepositContract.CallOpts)
}

// GenesisTime is a free data retrieval call binding the contract method 0x42c6498a.
//
// Solidity: function genesisTime() constant returns(bytes out)
func (_DepositContract *DepositContractCaller) GenesisTime(opts *bind.CallOpts) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "genesisTime")
	return *ret0, err
}

// GenesisTime is a free data retrieval call binding the contract method 0x42c6498a.
//
// Solidity: function genesisTime() constant returns(bytes out)
func (_DepositContract *DepositContractSession) GenesisTime() ([]byte, error) {
	return _DepositContract.Contract.GenesisTime(&_DepositContract.CallOpts)
}

// GenesisTime is a free data retrieval call binding the contract method 0x42c6498a.
//
// Solidity: function genesisTime() constant returns(bytes out)
func (_DepositContract *DepositContractCallerSession) GenesisTime() ([]byte, error) {
	return _DepositContract.Contract.GenesisTime(&_DepositContract.CallOpts)
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
