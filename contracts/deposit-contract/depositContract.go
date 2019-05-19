// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package depositContract

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
const DepositContractABI = "[{\"name\":\"Deposit\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"amount\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"signature\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"merkle_tree_index\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"Eth2Genesis\",\"inputs\":[{\"type\":\"bytes32\",\"name\":\"deposit_root\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"deposit_count\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"time\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"depositThreshold\"},{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"uint256\",\"name\":\"maxDeposit\"},{\"type\":\"uint256\",\"name\":\"customChainstartDelay\"},{\"type\":\"address\",\"name\":\"_drain_address\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"to_little_endian_64\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"value\"}],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":10123},{\"name\":\"from_little_endian_64\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[{\"type\":\"bytes\",\"name\":\"value\"}],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":6013},{\"name\":\"get_deposit_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":79281},{\"name\":\"get_deposit_count\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":17812},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\"},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\"},{\"type\":\"bytes\",\"name\":\"signature\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":483985},{\"name\":\"drain\",\"outputs\":[],\"inputs\":[],\"constant\":false,\"payable\":false,\"type\":\"function\",\"gas\":35853},{\"name\":\"CHAIN_START_FULL_DEPOSIT_THRESHOLD\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":693},{\"name\":\"MIN_DEPOSIT_AMOUNT\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":723},{\"name\":\"MAX_DEPOSIT_AMOUNT\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":753},{\"name\":\"deposit_count\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":783},{\"name\":\"full_deposit_count\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":813},{\"name\":\"chainStarted\",\"outputs\":[{\"type\":\"bool\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":843},{\"name\":\"custom_chainstart_delay\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":873},{\"name\":\"genesisTime\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":3126},{\"name\":\"drain_address\",\"outputs\":[{\"type\":\"address\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":933}]"

// DepositContractBin is the compiled bytecode used for deploying new contracts.
const DepositContractBin = `0x740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a05260a06119bf6101403934156100a157600080fd5b602060806119bf0160c03960c05160205181106100bd57600080fd5b506101405160005561016051600155610180516002556101a0516008556101c051600a556101e06000601f818352015b60006101e0516020811061010057600080fd5b600360c052602060c02001546020826102000101526020810190506101e0516020811061012c57600080fd5b600360c052602060c02001546020826102000101526020810190508061020052610200905080516020820120905060605160016101e051018060405190131561017457600080fd5b809190121561018257600080fd5b6020811061018f57600080fd5b600360c052602060c020015560605160016101e05101806040519013156101b557600080fd5b80919012156101c357600080fd5b602081106101d057600080fd5b600360c052602060c020015460605160016101e05101806040519013156101f657600080fd5b809190121561020457600080fd5b6020811061021157600080fd5b600460c052602060c02001555b81516001018083528114156100ed575b50506119a756600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a0526000156101a3575b6101605261014052601860086020820661018001602082840111156100bf57600080fd5b6020806101a082610140600060046015f1505081815280905090509050805160200180610240828460006004600a8704601201f16100fc57600080fd5b50506102405160206001820306601f82010390506102a0610240516008818352015b826102a051111561012e5761014a565b60006102a05161026001535b815160010180835281141561011e575b5050506020610220526040610240510160206001820306601f8201039050610200525b60006102005111151561017f5761019b565b602061020051036102200151602061020051036102005261016d565b610160515650005b638067328960005114156103c257602060046101403734156101c457600080fd5b67ffffffffffffffff6101405111156101dc57600080fd5b60006101605261014051610180526101a060006008818352015b6101605160086000811215610213578060000360020a820461021a565b8060020a82025b905090506101605260ff61018051166101c052610160516101c0516101605101101561024557600080fd5b6101c051610160510161016052610180517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8600081121561028e578060000360020a8204610295565b8060020a82025b90509050610180525b81516001018083528114156101f6575b505061014051610160516101805163b0429c706101e0526101605161020052610200516006580161009b565b506102605260006102c0525b6102605160206001820306601f82010390506102c05110151561030857610321565b6102c05161028001526102c0516020016102c0526102e6565b610180526101605261014052610260805160200180610320828460006004600a8704601201f161035057600080fd5b50506103205160206001820306601f8201039050610380610320516008818352015b826103805111156103825761039e565b60006103805161034001535b8151600101808352811415610372575b5050506020610300526040610320510160206001820306601f8201039050610300f3005b639d70e806600051141561055c57602060046101403734156103e357600080fd5b602860043560040161016037600860043560040135111561040357600080fd5b60006101c05261016080602001516000825180602090131561042457600080fd5b809190121561043257600080fd5b806020036101000a82049050905090506101e05261020060006008818352015b60ff6101e05116606051606051610200516007038060405190131561047657600080fd5b809190121561048457600080fd5b6008028060405190131561049757600080fd5b80919012156104a557600080fd5b60008112156104bc578060000360020a82046104c3565b8060020a82025b90509050610220526101c051610220516101c0510110156104e357600080fd5b610220516101c051016101c0526101e0517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8600081121561052c578060000360020a8204610533565b8060020a82025b905090506101e0525b8151600101808352811415610452575b50506101c05160005260206000f3005b63c5f2892f60005114156106b457341561057557600080fd5b6000610140526005546101605261018060006020818352015b600160016101605116141561060f57600061018051602081106105b057600080fd5b600460c052602060c02001546020826102200101526020810190506101405160208261022001015260208101905080610220526102209050602060c0825160208401600060025af161060157600080fd5b60c05190506101405261067d565b6000610140516020826101a0010152602081019050610180516020811061063557600080fd5b600360c052602060c02001546020826101a0010152602081019050806101a0526101a09050602060c0825160208401600060025af161067357600080fd5b60c0519050610140525b610160600261068b57600080fd5b60028151048152505b815160010180835281141561058e575b50506101405160005260206000f3005b63621fd130600051141561078a5734156106cd57600080fd5b60606101c060246380673289610140526005546101605261015c6000305af16106f557600080fd5b6101e0805160200180610260828460006004600a8704601201f161071857600080fd5b50506102605160206001820306601f82010390506102c0610260516008818352015b826102c051111561074a57610766565b60006102c05161028001535b815160010180835281141561073a575b5050506020610240526040610260510160206001820306601f8201039050610240f3005b63c47e300d600051141561152557606060046101403760506004356004016101a03760306004356004013511156107c057600080fd5b60406024356004016102203760206024356004013511156107e057600080fd5b608060443560040161028037606060443560040135111561080057600080fd5b633b9aca00610340526103405161081657600080fd5b6103405134046103205260015461032051101561083257600080fd5b6060610440602463806732896103c052610320516103e0526103dc6000305af161085b57600080fd5b610460805160200180610360828460006004600a8704601201f161087e57600080fd5b50506005546104a05260006104c05260026104e05261050060006020818352015b60006104e0516108ae57600080fd5b6104e0516104a05160016104a0510110156108c857600080fd5b60016104a051010618156108db57610947565b6104c06060516001825101806040519013156108f657600080fd5b809190121561090457600080fd5b8152506104e080511515610919576000610933565b600281516002835102041461092d57600080fd5b60028151025b8152505b815160010180835281141561089f575b505060006101a06030806020846105e001018260208501600060046016f15050805182019150506000601060208206610560016020828401111561098a57600080fd5b60208061058082610520600060046015f15050818152809050905090506010806020846105e001018260208501600060046013f1505080518201915050806105e0526105e09050602060c0825160208401600060025af16109ea57600080fd5b60c0519050610540526000600060406020820661068001610280518284011115610a1357600080fd5b6060806106a0826020602088068803016102800160006004601bf1505081815280905090509050602060c0825160208401600060025af1610a5357600080fd5b60c05190506020826108800101526020810190506000604060206020820661074001610280518284011115610a8757600080fd5b606080610760826020602088068803016102800160006004601bf150508181528090509050905060208060208461080001018260208501600060046015f15050805182019150506105205160208261080001015260208101905080610800526108009050602060c0825160208401600060025af1610b0457600080fd5b60c051905060208261088001015260208101905080610880526108809050602060c0825160208401600060025af1610b3b57600080fd5b60c051905061066052600060006105405160208261092001015260208101905061022060208060208461092001018260208501600060046015f150508051820191505080610920526109209050602060c0825160208401600060025af1610ba157600080fd5b60c0519050602082610aa00101526020810190506000610360600880602084610a2001018260208501600060046012f150508051820191505060006018602082066109a00160208284011115610bf657600080fd5b6020806109c082610520600060046015f1505081815280905090509050601880602084610a2001018260208501600060046014f150508051820191505061066051602082610a2001015260208101905080610a2052610a209050602060c0825160208401600060025af1610c6957600080fd5b60c0519050602082610aa001015260208101905080610aa052610aa09050602060c0825160208401600060025af1610ca057600080fd5b60c051905061090052610b2060006020818352015b6104c051610b20511215610d35576000610b205160208110610cd657600080fd5b600460c052602060c0200154602082610b4001015260208101905061090051602082610b4001015260208101905080610b4052610b409050602060c0825160208401600060025af1610d2757600080fd5b60c051905061090052610d3a565b610d4b565b5b8151600101808352811415610cb5575b5050610900516104c05160208110610d6257600080fd5b600460c052602060c02001556005805460018254011015610d8257600080fd5b60018154018155506020610c40600463c5f2892f610be052610bfc6000305af1610dab57600080fd5b610c4051610bc0526060610ce060246380673289610c60526104a051610c8052610c7c6000305af1610ddc57600080fd5b610d00805160200180610d40828460006004600a8704601201f1610dff57600080fd5b505060a0610dc052610dc051610e00526101a0805160200180610dc051610e0001828460006004600a8704601201f1610e3757600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516040818352015b83610da051101515610e7557610e92565b6000610da0516020850101535b8151600101808352811415610e64575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc052610dc051610e2052610220805160200180610dc051610e0001828460006004600a8704601201f1610ee957600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516020818352015b83610da051101515610f2757610f44565b6000610da0516020850101535b8151600101808352811415610f16575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc052610dc051610e4052610360805160200180610dc051610e0001828460006004600a8704601201f1610f9b57600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516020818352015b83610da051101515610fd957610ff6565b6000610da0516020850101535b8151600101808352811415610fc8575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc052610dc051610e6052610280805160200180610dc051610e0001828460006004600a8704601201f161104d57600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516060818352015b83610da05110151561108b576110a8565b6000610da0516020850101535b815160010180835281141561107a575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc052610dc051610e8052610d40805160200180610dc051610e0001828460006004600a8704601201f16110ff57600080fd5b5050610dc051610e00015160206001820306601f8201039050610dc051610e0001610da081516020818352015b83610da05110151561113d5761115a565b6000610da0516020850101535b815160010180835281141561112c575b505050506020610dc051610e00015160206001820306601f8201039050610dc0510101610dc0527fdc5fc95703516abd38fa03c3737ff3b52dc52347055c8028460fdf5bbe2f12ce610dc051610e00a1600254610320511015156115235760068054600182540110156111cc57600080fd5b600181540181555060005460065414156115225742610ec05242610ee052620151806111f757600080fd5b62015180610ee05106610ec051101561120f57600080fd5b42610ee0526201518061122157600080fd5b62015180610ee05106610ec051036202a30042610ec05242610ee0526201518061124a57600080fd5b62015180610ee05106610ec051101561126257600080fd5b42610ee0526201518061127457600080fd5b62015180610ee05106610ec0510301101561128e57600080fd5b6202a30042610ec05242610ee052620151806112a957600080fd5b62015180610ee05106610ec05110156112c157600080fd5b42610ee052620151806112d357600080fd5b62015180610ee05106610ec0510301610ea0526060610f8060246380673289610f0052600554610f2052610f1c6000305af161130e57600080fd5b610fa0805160200180610fe0828460006004600a8704601201f161133157600080fd5b505060606110c06024638067328961104052610ea0516110605261105c6000305af161135c57600080fd5b6110e0805160200180611120828460006004600a8704601201f161137f57600080fd5b5050610bc0516111e05260606111a0526111a05161120052610fe08051602001806111a0516111e001828460006004600a8704601201f16113bf57600080fd5b50506111a0516111e0015160206001820306601f82010390506111a0516111e00161118081516020818352015b83611180511015156113fd5761141a565b6000611180516020850101535b81516001018083528114156113ec575b5050505060206111a0516111e0015160206001820306601f82010390506111a05101016111a0526111a051611220526111208051602001806111a0516111e001828460006004600a8704601201f161147157600080fd5b50506111a0516111e0015160206001820306601f82010390506111a0516111e00161118081516020818352015b83611180511015156114af576114cc565b6000611180516020850101535b815160010180835281141561149e575b5050505060206111a0516111e0015160206001820306601f82010390506111a05101016111a0527f08b71ef3f1b58f7a23ffb82e27f12f0888c8403f1ceb0ea7ea26b274e2189d4c6111a0516111e0a160016007555b5b005b639890220b600051141561155957341561153e57600080fd5b60006000600060003031600a546000f161155757600080fd5b005b634a637042600051141561157f57341561157257600080fd5b60005460005260206000f3005b631ea30fef60005114156115a557341561159857600080fd5b60015460005260206000f3005b634c34a98260005114156115cb5734156115be57600080fd5b60025460005260206000f3005b63eb8545ee60005114156115f15734156115e457600080fd5b60055460005260206000f3005b63188e6c87600051141561161757341561160a57600080fd5b60065460005260206000f3005b63845980e8600051141561163d57341561163057600080fd5b60075460005260206000f3005b63b6080cd2600051141561166357341561165657600080fd5b60085460005260206000f3005b6342c6498a600051141561174657341561167c57600080fd5b60098060c052602060c020610180602082540161012060006002818352015b826101205160200211156116ae576116d0565b61012051850154610120516020028501525b815160010180835281141561169b575b5050505050506101805160206001820306601f82010390506101e0610180516008818352015b826101e051111561170657611722565b60006101e0516101a001535b81516001018083528114156116f6575b5050506020610160526040610180510160206001820306601f8201039050610160f3005b638ba35cdf600051141561176c57341561175f57600080fd5b600a5460005260206000f3005b60006000fd5b6102356119a7036102356000396102356119a7036000f3`

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

// FromLittleEndian64 is a free data retrieval call binding the contract method 0x9d70e806.
//
// Solidity: function from_little_endian_64(bytes value) constant returns(uint256 out)
func (_DepositContract *DepositContractCaller) FromLittleEndian64(opts *bind.CallOpts, value []byte) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _DepositContract.contract.Call(opts, out, "from_little_endian_64", value)
	return *ret0, err
}

// FromLittleEndian64 is a free data retrieval call binding the contract method 0x9d70e806.
//
// Solidity: function from_little_endian_64(bytes value) constant returns(uint256 out)
func (_DepositContract *DepositContractSession) FromLittleEndian64(value []byte) (*big.Int, error) {
	return _DepositContract.Contract.FromLittleEndian64(&_DepositContract.CallOpts, value)
}

// FromLittleEndian64 is a free data retrieval call binding the contract method 0x9d70e806.
//
// Solidity: function from_little_endian_64(bytes value) constant returns(uint256 out)
func (_DepositContract *DepositContractCallerSession) FromLittleEndian64(value []byte) (*big.Int, error) {
	return _DepositContract.Contract.FromLittleEndian64(&_DepositContract.CallOpts, value)
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
	Pubkey                []byte
	WithdrawalCredentials []byte
	Amount                []byte
	Signature             []byte
	MerkleTreeIndex       []byte
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xdc5fc95703516abd38fa03c3737ff3b52dc52347055c8028460fdf5bbe2f12ce.
//
// Solidity: event Deposit(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes merkle_tree_index)
func (_DepositContract *DepositContractFilterer) FilterDeposit(opts *bind.FilterOpts) (*DepositContractDepositIterator, error) {

	logs, sub, err := _DepositContract.contract.FilterLogs(opts, "Deposit")
	if err != nil {
		return nil, err
	}
	return &DepositContractDepositIterator{contract: _DepositContract.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xdc5fc95703516abd38fa03c3737ff3b52dc52347055c8028460fdf5bbe2f12ce.
//
// Solidity: event Deposit(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes merkle_tree_index)
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

// DepositContractEth2GenesisIterator is returned from FilterEth2Genesis and is used to iterate over the raw logs and unpacked data for Eth2Genesis events raised by the DepositContract contract.
type DepositContractEth2GenesisIterator struct {
	Event *DepositContractEth2Genesis // Event containing the contract specifics and raw log

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
func (it *DepositContractEth2GenesisIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DepositContractEth2Genesis)
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
		it.Event = new(DepositContractEth2Genesis)
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
func (it *DepositContractEth2GenesisIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DepositContractEth2GenesisIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DepositContractEth2Genesis represents a Eth2Genesis event raised by the DepositContract contract.
type DepositContractEth2Genesis struct {
	DepositRoot  [32]byte
	DepositCount []byte
	Time         []byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterEth2Genesis is a free log retrieval operation binding the contract event 0x08b71ef3f1b58f7a23ffb82e27f12f0888c8403f1ceb0ea7ea26b274e2189d4c.
//
// Solidity: event Eth2Genesis(bytes32 deposit_root, bytes deposit_count, bytes time)
func (_DepositContract *DepositContractFilterer) FilterEth2Genesis(opts *bind.FilterOpts) (*DepositContractEth2GenesisIterator, error) {

	logs, sub, err := _DepositContract.contract.FilterLogs(opts, "Eth2Genesis")
	if err != nil {
		return nil, err
	}
	return &DepositContractEth2GenesisIterator{contract: _DepositContract.contract, event: "Eth2Genesis", logs: logs, sub: sub}, nil
}

// WatchEth2Genesis is a free log subscription operation binding the contract event 0x08b71ef3f1b58f7a23ffb82e27f12f0888c8403f1ceb0ea7ea26b274e2189d4c.
//
// Solidity: event Eth2Genesis(bytes32 deposit_root, bytes deposit_count, bytes time)
func (_DepositContract *DepositContractFilterer) WatchEth2Genesis(opts *bind.WatchOpts, sink chan<- *DepositContractEth2Genesis) (event.Subscription, error) {

	logs, sub, err := _DepositContract.contract.WatchLogs(opts, "Eth2Genesis")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DepositContractEth2Genesis)
				if err := _DepositContract.contract.UnpackLog(event, "Eth2Genesis", log); err != nil {
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
