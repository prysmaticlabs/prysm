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
const DepositContractABI = "[{\"name\":\"DepositEvent\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"amount\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"signature\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"index\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"outputs\":[],\"inputs\":[{\"type\":\"address\",\"name\":\"_drain_address\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"get_deposit_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":95727},{\"name\":\"get_deposit_count\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":18283},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\"},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\"},{\"type\":\"bytes\",\"name\":\"signature\"},{\"type\":\"bytes32\",\"name\":\"deposit_data_root\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":1342680},{\"name\":\"drain\",\"outputs\":[],\"inputs\":[],\"constant\":false,\"payable\":false,\"type\":\"function\",\"gas\":35831},{\"name\":\"drain_address\",\"outputs\":[{\"type\":\"address\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":701}]"

// DepositContractBin is the compiled bytecode used for deploying new contracts.
var DepositContractBin = "0x740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a05260206113166101403934156100a157600080fd5b602061131660c03960c05160205181106100ba57600080fd5b50610140516002556101606000601f818352015b600061016051602081106100e157600080fd5b600360c052602060c0200154602082610180010152602081019050610160516020811061010d57600080fd5b600360c052602060c020015460208261018001015260208101905080610180526101809050602060c0825160208401600060025af161014b57600080fd5b60c0519050606051600161016051018060405190131561016a57600080fd5b809190121561017857600080fd5b6020811061018557600080fd5b600360c052602060c02001555b81516001018083528114156100ce575b50506112fe56600436101561000d5761114f565b600035601c52740100000000000000000000000000000000000000006020526f7fffffffffffffffffffffffffffffff6040527fffffffffffffffffffffffffffffffff8000000000000000000000000000000060605274012a05f1fffffffffffffffffffffffffdabf41c006080527ffffffffffffffffffffffffed5fa0e000000000000000000000000000000000060a052600015610286575b6101605261014052600061018052610140516101a0526101c060006008818352015b61018051600860008112156100e8578060000360020a82046100ef565b8060020a82025b905090506101805260ff6101a051166101e052610180516101e0516101805101101561011a57600080fd5b6101e0516101805101610180526101a0517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff86000811215610163578060000360020a820461016a565b8060020a82025b905090506101a0525b81516001018083528114156100cb575b5050601860086020820661020001602082840111156101a157600080fd5b60208061022082610180600060046015f15050818152809050905090508051602001806102c0828460006004600a8704601201f16101de57600080fd5b50506102c05160206001820306601f82010390506103206102c0516020818352015b82610320511015156102115761022d565b6000610320516102e001535b8151600101808352811415610200575b50505060206102a05260406102c0510160206001820306601f8201039050610280525b6000610280511115156102625761027e565b602061028051036102a001516020610280510361028052610250565b610160515650005b63c5f2892f600051141561051857341561029f57600080fd5b6000610140526101405161016052600154610180526101a060006020818352015b60016001610180511614156103415760006101a051602081106102e257600080fd5b600060c052602060c02001546020826102400101526020810190506101605160208261024001015260208101905080610240526102409050602060c0825160208401600060025af161033357600080fd5b60c0519050610160526103af565b6000610160516020826101c00101526020810190506101a0516020811061036757600080fd5b600360c052602060c02001546020826101c0010152602081019050806101c0526101c09050602060c0825160208401600060025af16103a557600080fd5b60c0519050610160525b61018060026103bd57600080fd5b60028151048152505b81516001018083528114156102c0575b505060006101605160208261046001015260208101905061014051610160516101805163806732896102e0526001546103005261030051600658016100a9565b506103605260006103c0525b6103605160206001820306601f82010390506103c0511015156104445761045d565b6103c05161038001526103c0516020016103c052610422565b61018052610160526101405261036060088060208461046001018260208501600060046012f150508051820191505060006018602082066103e001602082840111156104a857600080fd5b60208061040082610140600060046015f150508181528090509050905060188060208461046001018260208501600060046014f150508051820191505080610460526104609050602060c0825160208401600060025af161050857600080fd5b60c051905060005260206000f350005b63621fd130600051141561062c57341561053157600080fd5b6380673289610140526001546101605261016051600658016100a9565b506101c0526000610220525b6101c05160206001820306601f82010390506102205110151561057c57610595565b610220516101e00152610220516020016102205261055a565b6101c0805160200180610280828460006004600a8704601201f16105b857600080fd5b50506102805160206001820306601f82010390506102e0610280516020818352015b826102e0511015156105eb57610607565b60006102e0516102a001535b81516001018083528114156105da575b5050506020610260526040610280510160206001820306601f8201039050610260f350005b632289511860005114156110f357605060043560040161014037603060043560040135111561065a57600080fd5b60406024356004016101c037602060243560040135111561067a57600080fd5b608060443560040161022037606060443560040135111561069a57600080fd5b63ffffffff600154106106ac57600080fd5b633b9aca006102e0526102e0516106c257600080fd5b6102e05134046102c052633b9aca006102c05110156106e057600080fd5b603061014051146106f057600080fd5b60206101c0511461070057600080fd5b6060610220511461071057600080fd5b610140610360525b6103605151602061036051016103605261036061036051101561073a57610718565b6380673289610380526102c0516103a0526103a051600658016100a9565b50610400526000610460525b6104005160206001820306601f8201039050610460511015156107865761079f565b6104605161042001526104605160200161046052610764565b610340610360525b61036051526020610360510361036052610140610360511015156107ca576107a7565b610400805160200180610300828460006004600a8704601201f16107ed57600080fd5b5050610140610480525b61048051516020610480510161048052610480610480511015610819576107f7565b63806732896104a0526001546104c0526104c051600658016100a9565b50610520526000610580525b6105205160206001820306601f8201039050610580511015156108645761087d565b6105805161054001526105805160200161058052610842565b610460610480525b61048051526020610480510361048052610140610480511015156108a857610885565b6105208051602001806105a0828460006004600a8704601201f16108cb57600080fd5b505060a06106205261062051610660526101408051602001806106205161066001828460006004600a8704601201f161090357600080fd5b505061062051610660015160206001820306601f82010390506106006106205161066001516040818352015b826106005110151561094057610961565b600061060051610620516106800101535b815160010180835281141561092f575b505050602061062051610660015160206001820306601f82010390506106205101016106205261062051610680526101c08051602001806106205161066001828460006004600a8704601201f16109b757600080fd5b505061062051610660015160206001820306601f82010390506106006106205161066001516020818352015b82610600511015156109f457610a15565b600061060051610620516106800101535b81516001018083528114156109e3575b505050602061062051610660015160206001820306601f820103905061062051010161062052610620516106a0526103008051602001806106205161066001828460006004600a8704601201f1610a6b57600080fd5b505061062051610660015160206001820306601f82010390506106006106205161066001516020818352015b8261060051101515610aa857610ac9565b600061060051610620516106800101535b8151600101808352811415610a97575b505050602061062051610660015160206001820306601f820103905061062051010161062052610620516106c0526102208051602001806106205161066001828460006004600a8704601201f1610b1f57600080fd5b505061062051610660015160206001820306601f82010390506106006106205161066001516060818352015b8261060051101515610b5c57610b7d565b600061060051610620516106800101535b8151600101808352811415610b4b575b505050602061062051610660015160206001820306601f820103905061062051010161062052610620516106e0526105a08051602001806106205161066001828460006004600a8704601201f1610bd357600080fd5b505061062051610660015160206001820306601f82010390506106006106205161066001516020818352015b8261060051101515610c1057610c31565b600061060051610620516106800101535b8151600101808352811415610bff575b505050602061062051610660015160206001820306601f8201039050610620510101610620527f649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c561062051610660a160006107005260006101406030806020846107c001018260208501600060046016f150508051820191505060006010602082066107400160208284011115610cc757600080fd5b60208061076082610700600060046015f15050818152809050905090506010806020846107c001018260208501600060046013f1505080518201915050806107c0526107c09050602060c0825160208401600060025af1610d2757600080fd5b60c0519050610720526000600060406020820661086001610220518284011115610d5057600080fd5b606080610880826020602088068803016102200160006004601bf1505081815280905090509050602060c0825160208401600060025af1610d9057600080fd5b60c0519050602082610a600101526020810190506000604060206020820661092001610220518284011115610dc457600080fd5b606080610940826020602088068803016102200160006004601bf15050818152809050905090506020806020846109e001018260208501600060046015f1505080518201915050610700516020826109e0010152602081019050806109e0526109e09050602060c0825160208401600060025af1610e4157600080fd5b60c0519050602082610a6001015260208101905080610a6052610a609050602060c0825160208401600060025af1610e7857600080fd5b60c0519050610840526000600061072051602082610b000101526020810190506101c0602080602084610b0001018260208501600060046015f150508051820191505080610b0052610b009050602060c0825160208401600060025af1610ede57600080fd5b60c0519050602082610c800101526020810190506000610300600880602084610c0001018260208501600060046012f15050805182019150506000601860208206610b800160208284011115610f3357600080fd5b602080610ba082610700600060046015f1505081815280905090509050601880602084610c0001018260208501600060046014f150508051820191505061084051602082610c0001015260208101905080610c0052610c009050602060c0825160208401600060025af1610fa657600080fd5b60c0519050602082610c8001015260208101905080610c8052610c809050602060c0825160208401600060025af1610fdd57600080fd5b60c0519050610ae052606435610ae05114610ff757600080fd5b600180546001825401101561100b57600080fd5b6001815401815550600154610d0052610d2060006020818352015b60016001610d005116141561105b57610ae051610d20516020811061104a57600080fd5b600060c052602060c02001556110ef565b6000610d20516020811061106e57600080fd5b600060c052602060c0200154602082610d40010152602081019050610ae051602082610d4001015260208101905080610d4052610d409050602060c0825160208401600060025af16110bf57600080fd5b60c0519050610ae052610d0060026110d657600080fd5b60028151048152505b8151600101808352811415611026575b5050005b639890220b600051141561112757341561110c57600080fd5b600060006000600030316002546000f161112557600080fd5b005b638ba35cdf600051141561114e57341561114057600080fd5b60025460005260206000f350005b5b60006000fd5b6101a96112fe036101a96000396101a96112fe036000f3"

// DeployDepositContract deploys a new Ethereum contract, binding an instance of DepositContract to it.
func DeployDepositContract(auth *bind.TransactOpts, backend bind.ContractBackend, _drain_address common.Address) (common.Address, *types.Transaction, *DepositContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DepositContractBin), backend, _drain_address)
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

// Deposit is a paid mutator transaction binding the contract method 0x22895118.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature, bytes32 deposit_data_root) returns()
func (_DepositContract *DepositContractTransactor) Deposit(opts *bind.TransactOpts, pubkey []byte, withdrawal_credentials []byte, signature []byte, deposit_data_root [32]byte) (*types.Transaction, error) {
	return _DepositContract.contract.Transact(opts, "deposit", pubkey, withdrawal_credentials, signature, deposit_data_root)
}

// Deposit is a paid mutator transaction binding the contract method 0x22895118.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature, bytes32 deposit_data_root) returns()
func (_DepositContract *DepositContractSession) Deposit(pubkey []byte, withdrawal_credentials []byte, signature []byte, deposit_data_root [32]byte) (*types.Transaction, error) {
	return _DepositContract.Contract.Deposit(&_DepositContract.TransactOpts, pubkey, withdrawal_credentials, signature, deposit_data_root)
}

// Deposit is a paid mutator transaction binding the contract method 0x22895118.
//
// Solidity: function deposit(bytes pubkey, bytes withdrawal_credentials, bytes signature, bytes32 deposit_data_root) returns()
func (_DepositContract *DepositContractTransactorSession) Deposit(pubkey []byte, withdrawal_credentials []byte, signature []byte, deposit_data_root [32]byte) (*types.Transaction, error) {
	return _DepositContract.Contract.Deposit(&_DepositContract.TransactOpts, pubkey, withdrawal_credentials, signature, deposit_data_root)
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

// ParseDepositEvent is a log parse operation binding the contract event 0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5.
//
// Solidity: event DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
func (_DepositContract *DepositContractFilterer) ParseDepositEvent(log types.Log) (*DepositContractDepositEvent, error) {
	event := new(DepositContractDepositEvent)
	if err := _DepositContract.contract.UnpackLog(event, "DepositEvent", log); err != nil {
		return nil, err
	}
	return event, nil
}
