// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// RLPABI is the input ABI used to generate the binding from.
const RLPABI = "[]"

// RLPBin is the compiled bytecode used for deploying new contracts.
const RLPBin = `0x60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058207370f347c211b0e9f613e9f0258468c5c9bece58db1173ee5351746458418c540029`

// DeployRLP deploys a new Ethereum contract, binding an instance of RLP to it.
func DeployRLP(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *RLP, error) {
	parsed, err := abi.JSON(strings.NewReader(RLPABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(RLPBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &RLP{RLPCaller: RLPCaller{contract: contract}, RLPTransactor: RLPTransactor{contract: contract}}, nil
}

// RLP is an auto generated Go binding around an Ethereum contract.
type RLP struct {
	RLPCaller     // Read-only binding to the contract
	RLPTransactor // Write-only binding to the contract
}

// RLPCaller is an auto generated read-only Go binding around an Ethereum contract.
type RLPCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLPTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RLPTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLPSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RLPSession struct {
	Contract     *RLP              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RLPCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RLPCallerSession struct {
	Contract *RLPCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// RLPTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RLPTransactorSession struct {
	Contract     *RLPTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RLPRaw is an auto generated low-level Go binding around an Ethereum contract.
type RLPRaw struct {
	Contract *RLP // Generic contract binding to access the raw methods on
}

// RLPCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RLPCallerRaw struct {
	Contract *RLPCaller // Generic read-only contract binding to access the raw methods on
}

// RLPTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RLPTransactorRaw struct {
	Contract *RLPTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRLP creates a new instance of RLP, bound to a specific deployed contract.
func NewRLP(address common.Address, backend bind.ContractBackend) (*RLP, error) {
	contract, err := bindRLP(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RLP{RLPCaller: RLPCaller{contract: contract}, RLPTransactor: RLPTransactor{contract: contract}}, nil
}

// NewRLPCaller creates a new read-only instance of RLP, bound to a specific deployed contract.
func NewRLPCaller(address common.Address, caller bind.ContractCaller) (*RLPCaller, error) {
	contract, err := bindRLP(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &RLPCaller{contract: contract}, nil
}

// NewRLPTransactor creates a new write-only instance of RLP, bound to a specific deployed contract.
func NewRLPTransactor(address common.Address, transactor bind.ContractTransactor) (*RLPTransactor, error) {
	contract, err := bindRLP(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &RLPTransactor{contract: contract}, nil
}

// bindRLP binds a generic wrapper to an already deployed contract.
func bindRLP(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(RLPABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RLP *RLPRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _RLP.Contract.RLPCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RLP *RLPRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RLP.Contract.RLPTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RLP *RLPRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RLP.Contract.RLPTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RLP *RLPCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _RLP.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RLP *RLPTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RLP.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RLP *RLPTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RLP.Contract.contract.Transact(opts, method, params...)
}

// SigHasherContractABI is the input ABI used to generate the binding from.
const SigHasherContractABI = "[]"

// SigHasherContractBin is the compiled bytecode used for deploying new contracts.
const SigHasherContractBin = `0x60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a72305820d1e8d7c8f2fa07496a4b23f8a6d60871008288d5fac673be890f35954e859d0f0029`

// DeploySigHasherContract deploys a new Ethereum contract, binding an instance of SigHasherContract to it.
func DeploySigHasherContract(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SigHasherContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SigHasherContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SigHasherContractBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SigHasherContract{SigHasherContractCaller: SigHasherContractCaller{contract: contract}, SigHasherContractTransactor: SigHasherContractTransactor{contract: contract}}, nil
}

// SigHasherContract is an auto generated Go binding around an Ethereum contract.
type SigHasherContract struct {
	SigHasherContractCaller     // Read-only binding to the contract
	SigHasherContractTransactor // Write-only binding to the contract
}

// SigHasherContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type SigHasherContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SigHasherContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SigHasherContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SigHasherContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SigHasherContractSession struct {
	Contract     *SigHasherContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// SigHasherContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SigHasherContractCallerSession struct {
	Contract *SigHasherContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// SigHasherContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SigHasherContractTransactorSession struct {
	Contract     *SigHasherContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// SigHasherContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type SigHasherContractRaw struct {
	Contract *SigHasherContract // Generic contract binding to access the raw methods on
}

// SigHasherContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SigHasherContractCallerRaw struct {
	Contract *SigHasherContractCaller // Generic read-only contract binding to access the raw methods on
}

// SigHasherContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SigHasherContractTransactorRaw struct {
	Contract *SigHasherContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSigHasherContract creates a new instance of SigHasherContract, bound to a specific deployed contract.
func NewSigHasherContract(address common.Address, backend bind.ContractBackend) (*SigHasherContract, error) {
	contract, err := bindSigHasherContract(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SigHasherContract{SigHasherContractCaller: SigHasherContractCaller{contract: contract}, SigHasherContractTransactor: SigHasherContractTransactor{contract: contract}}, nil
}

// NewSigHasherContractCaller creates a new read-only instance of SigHasherContract, bound to a specific deployed contract.
func NewSigHasherContractCaller(address common.Address, caller bind.ContractCaller) (*SigHasherContractCaller, error) {
	contract, err := bindSigHasherContract(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &SigHasherContractCaller{contract: contract}, nil
}

// NewSigHasherContractTransactor creates a new write-only instance of SigHasherContract, bound to a specific deployed contract.
func NewSigHasherContractTransactor(address common.Address, transactor bind.ContractTransactor) (*SigHasherContractTransactor, error) {
	contract, err := bindSigHasherContract(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &SigHasherContractTransactor{contract: contract}, nil
}

// bindSigHasherContract binds a generic wrapper to an already deployed contract.
func bindSigHasherContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SigHasherContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SigHasherContract *SigHasherContractRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SigHasherContract.Contract.SigHasherContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SigHasherContract *SigHasherContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SigHasherContract.Contract.SigHasherContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SigHasherContract *SigHasherContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SigHasherContract.Contract.SigHasherContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SigHasherContract *SigHasherContractCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SigHasherContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SigHasherContract *SigHasherContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SigHasherContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SigHasherContract *SigHasherContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SigHasherContract.Contract.contract.Transact(opts, method, params...)
}

// VMCABI is the input ABI used to generate the binding from.
const VMCABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_validatorIndex\",\"type\":\"int256\"},{\"name\":\"_sig\",\"type\":\"bytes32\"}],\"name\":\"withdraw\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getValidatorsMaxIndex\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_txStartgas\",\"type\":\"uint256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes12\"}],\"name\":\"txToShard\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"getAncestorDistance\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_valcodeAddr\",\"type\":\"address\"}],\"name\":\"getShardList\",\"outputs\":[{\"name\":\"\",\"type\":\"bool[100]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getCollationGasLimit\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_expectedPeriodNumber\",\"type\":\"uint256\"}],\"name\":\"getPeriodStartPrevhash\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"int256\"}],\"name\":\"sample\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_receiptId\",\"type\":\"int256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"}],\"name\":\"updataGasPrice\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_header\",\"type\":\"bytes\"}],\"name\":\"addHeader\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_validationCodeAddr\",\"type\":\"address\"},{\"name\":\"_returnAddr\",\"type\":\"address\"}],\"name\":\"deposit\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// VMCBin is the compiled bytecode used for deploying new contracts.
const VMCBin = `0x6060604052341561000f57600080fd5b6000600481905560075568056bc75e2d631000006008556005600981905562061a80600a55600c556064600d819055600e556040517f6164645f686561646572282900000000000000000000000000000000000000008152600c01604051908190039020600f5560108054600160a060020a03191673dffd41e18f04ad8810c83b14fd1426a82e625a7d1790556113f5806100ab6000396000f3006060604052600436106100ae5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630e3b2f0e81146100b35780632b3407f9146100e0578063372a9e2a146101055780635badac531461012f5780635e57c86c14610145578063934586ec1461019d5780639b33f907146101b0578063a8c57753146101c6578063e551e00a146101f8578063f7fecf7d14610206578063f9609f0814610257575b600080fd5b34156100be57600080fd5b6100cc600435602435610271565b604051901515815260200160405180910390f35b34156100eb57600080fd5b6100f3610414565b60405190815260200160405180910390f35b6100f3600160a060020a0360043516602435604435606435600160a060020a0319608435166104bb565b341561013a57600080fd5b6100f36004356105f7565b341561015057600080fd5b610164600160a060020a03600435166105fd565b6040518082610c8080838360005b8381101561018a578082015183820152602001610172565b5050505090500191505060405180910390f35b34156101a857600080fd5b6100f361073b565b34156101bb57600080fd5b6100f3600435610743565b34156101d157600080fd5b6101dc600435610767565b604051600160a060020a03909116815260200160405180910390f35b6100cc60043560243561087e565b341561021157600080fd5b6100cc60046024813581810190830135806020601f820181900481020160405190810160405281815292919060208401838380828437509496506108c395505050505050565b6100f3600160a060020a0360043581169060243516610c61565b60008060006040517f776974686472617700000000000000000000000000000000000000000000000081526008016040519081900390206000868152602081905260409081902060010154600a54929450600160a060020a031691638e19899e918790517c010000000000000000000000000000000000000000000000000000000063ffffffff85160281526004810191909152602401602060405180830381600088803b151561032157600080fd5b87f1151561032e57600080fd5b505050506040518051915050801561040c576000858152602081905260409081902060028101549054600160a060020a039091169181156108fc02919051600060405180830381858888f19350505050151561038957600080fd5b600085815260208181526040808320600181018054600160a060020a03168552600b8452918420805460ff19169055888452918390528282558054600160a060020a03199081169091556002820180549091169055600301556103eb85610de8565b60048054600019019055848260405190815260200160405180910390a18092505b505092915050565b60008060008060008060009450600093506009544381151561043257fe5b059250600754600454019150600090505b6104008112156104ac57818112610459576104ac565b600081815260208190526040902060010154600160a060020a038681169116148015906104985750600081815260208190526040902060030154839013155b156104a4576001840193505b600101610443565b60075484019550505050505090565b60008060e060405190810160409081528782526020808301889052818301879052346060840152600160a060020a0333811660808501528a1660a0840152600160a060020a0319861660c08401526005546000908152600290915220815181556020820151816001015560408201518160020155606082015181600301556080820151600482018054600160a060020a031916600160a060020a039290921691909117905560a0820151600582018054600160a060020a031916600160a060020a039290921691909117905560c0820151600690910155505060058054600181019091558086600160a060020a0389166040517f74785f746f5f73686172642829000000000000000000000000000000000000008152600d01604051809103902060405190815260200160405180910390a39695505050505050565b50600090565b6106056112fb565b61060d6112fb565b60008060008060008060006009544381151561062557fe5b05965060016009548802039550600086121561064057600095505b8540945061064c610414565b6004549094501561072a57600092505b60648360ff16101561072a5760008860ff85166064811061067957fe5b91151560209092020152600091505b60648260ff16101561071f57838560ff8086169085166040519283526020830191909152604080830191909152606090910190519081900390208115156106cb57fe5b06600081815260208190526040902060010154909150600160a060020a038b8116911614156107145760018860ff85166064811061070557fe5b9115156020909202015261071f565b816001019150610688565b82600101925061065c565b8798505b5050505050505050919050565b629896805b90565b600c546000908202600019014381901161075c57600080fd5b804091505b50919050565b6000806000806000806000600c54431015151561078357600080fd5b6009544381151561079057fe5b0595506001600954870203945060008512156107ab57600094505b600c5485409450600190438115156107bf57fe5b0643030340600190049250600d5483896001026040519182526020820152604090810190519081900390208115156107f357fe5b0691506107fe610414565b84898460405192835260208301919091526040808301919091526060909101905190819003902081151561082e57fe5b06600081815260208190526040902060030154909150869013156108555760009650610873565b600081815260208190526040902060010154600160a060020a031696505b505050505050919050565b60008281526002602052604081206004015433600160a060020a039081169116146108a857600080fd5b50600091825260026020819052604090922090910155600190565b6000806108ce611324565b6108d6611336565b6108de611357565b60009350859250838080806109026108fd88600163ffffffff610e0716565b610e6b565b95506101406040519081016040528061092261091d89610ea4565b610ee9565b815260200161093861093389610ea4565b610efa565b815260200161094961091d89610ea4565b815260200161095a61091d89610ea4565b815260200161096b61091d89610ea4565b815260200161098161097c89610ea4565b610f4b565b600160a060020a0316815260200161099b61091d89610ea4565b81526020016109ac61091d89610ea4565b81526020016109bd61091d89610ea4565b81526020016109d36109ce89610ea4565b610f95565b9052945060008551121580156109eb5750600e548551125b15156109f657600080fd5b600c54431015610a0557600080fd5b600c5443811515610a1257fe5b04856020015114610a2257600080fd5b6001600c548660200151020340604086015114610a3e57600080fd5b846040519081526020016040519081900390209350831515610a5c57fe5b60016000865181526020808201929092526040908101600090812087825290925290206001015415610a8a57fe5b846060015115610ada5784606001511580610ad25750600060018187518152602001908152602001600020600087606001518152602081019190915260400160002060010154135b1515610ada57fe5b8460200151601160008751815260200190815260200160002054121515610afd57fe5b610b078551610767565b9250600160a060020a0383161515610b22576000985061072e565b600160008651815260200190815260200160002060008660600151815260208101919091526040016000206001908101540191508161010086015114610b6457fe5b6040805190810160405280866060015181526020018390526001600087518152602080820192909252604090810160009081208882529092529020815181556020820151600190910155506020850151601160008751815260200190815260200160002081905550600160008660000151815260200190815260200160002060006003600088600001518152602001908152602001600020546000191660001916815260200190815260200160002060010154821315610c515760036000865181526020019081526020016000205490508360036000876000015181526020810191909152604001600020555b5060019998505050505050505050565b600160a060020a0382166000908152600b60205260408120548190819060ff1615610c8b57600080fd5b6008543414610c9957600080fd5b506000610ca4610fe3565b1515610cb957610cb2610fea565b9150610d6c565b600454915060095443811515610ccb57fe5b05600101905060806040519081016040908152348252600160a060020a03808816602080850191909152908716828401526060830184905260008581529081905220815181556020820151600182018054600160a060020a031916600160a060020a03929092169190911790556040820151600282018054600160a060020a031916600160a060020a03929092169190911790556060820151600390910155505b600480546001908101909155600160a060020a0386166000818152600b602052604090819020805460ff19169093179092558391517f6465706f736974282900000000000000000000000000000000000000000000008152600901604051809103902060405190815260200160405180910390a2509392505050565b6007805460009081526006602052604090209190915580546001019055565b610e0f6113b2565b610e176113b2565b6000610e2285611021565b91508315610e63578451905080610e3883611072565b1115610e4057fe5b80610e4b83516110f1565b14610e5257fe5b610e5b82611183565b1515610e6357fe5b509392505050565b610e73611336565b6000610e7e836111c5565b1515610e8957600080fd5b610e9283611072565b83519383529092016020820152919050565b610eac6113b2565b600080610eb8846111f0565b156100ae5783602001519150610ecd826110f1565b82845260208085018290528382019086015290505b5050919050565b6000610ef482610efa565b92915050565b6000806000610f0884611213565b1515610f1357600080fd5b610f1c8461123d565b9150915060208111158015610f3057508015155b1515610f3857fe5b806020036101000a825104949350505050565b6000806000610f5984611213565b1515610f6457600080fd5b610f6d8461123d565b909250905060148114610f7c57fe5b6c01000000000000000000000000825104949350505050565b610f9d611324565b600082602001519050801515610fb257610761565b80604051805910610fc05750595b818152601f19601f830116810160200160405290509150610761835183836112ba565b6007541590565b6000610ff4610fe3565b156110025750600019610740565b5060078054600019019081905560009081526006602052604090205490565b6110296113b2565b600080835191508115156110525760408051908101604052600080825260208201529250610ee2565b506020830160408051908101604052908152602081019190915292915050565b60008060008360200151151561108b5760009250610ee2565b83519050805160001a915060808210156110a85760009250610ee2565b60b88210806110c3575060c082101580156110c3575060f882105b156110d15760019250610ee2565b60c08210156110e65760b51982019250610ee2565b5060f5190192915050565b600080825160001a9050608081101561110d5760019150610761565b60b881101561112257607e1981019150610761565b60c081101561114c5760b78103806020036101000a60018501510480820160010193505050610761565b60f88110156111615760be1981019150610761565b60f78103806020036101000a6001850151048082016001019350505050919050565b600080808084519050805160001a9250805160011a91506081831480156111aa5750608082105b156111b857600093506111bd565b600193505b505050919050565b600080826020015115156111dc5760009150610761565b8251905060c0815160001a10159392505050565b60006111fa6113b2565b8251905080602001518151018360200151109392505050565b6000808260200151151561122a5760009150610761565b8251905060c0815160001a109392505050565b600080600080600061124e86611213565b151561125957600080fd5b85519150815160001a9250608083101561127957819450600193506112b2565b60b883101561129757600186602001510393508160010194506112b2565b5060b619820180600160208801510303935080820160010194505b505050915091565b60006020601f83010484602085015b8284146112e857602084028083015181830152600185019450506112c9565b6000865160200187015250505050505050565b610c806040519081016040526064815b60008152600019909101906020018161130b5790505090565b60206040519081016040526000815290565b60606040519081016040528061134a6113b2565b8152602001600081525090565b6101406040519081016040908152600080835260208301819052908201819052606082018190526080820181905260a0820181905260c0820181905260e0820181905261010082015261012081016113ad611324565b905290565b6040805190810160405260008082526020820152905600a165627a7a72305820ac752c2e2510c5d4616642d59dff6585ad73413dba273192b11758fd542c54ec0029`

// DeployVMC deploys a new Ethereum contract, binding an instance of VMC to it.
func DeployVMC(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *VMC, error) {
	parsed, err := abi.JSON(strings.NewReader(VMCABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(VMCBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &VMC{VMCCaller: VMCCaller{contract: contract}, VMCTransactor: VMCTransactor{contract: contract}}, nil
}

// VMC is an auto generated Go binding around an Ethereum contract.
type VMC struct {
	VMCCaller     // Read-only binding to the contract
	VMCTransactor // Write-only binding to the contract
}

// VMCCaller is an auto generated read-only Go binding around an Ethereum contract.
type VMCCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VMCTransactor is an auto generated write-only Go binding around an Ethereum contract.
type VMCTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VMCSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type VMCSession struct {
	Contract     *VMC              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VMCCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type VMCCallerSession struct {
	Contract *VMCCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// VMCTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type VMCTransactorSession struct {
	Contract     *VMCTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VMCRaw is an auto generated low-level Go binding around an Ethereum contract.
type VMCRaw struct {
	Contract *VMC // Generic contract binding to access the raw methods on
}

// VMCCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type VMCCallerRaw struct {
	Contract *VMCCaller // Generic read-only contract binding to access the raw methods on
}

// VMCTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type VMCTransactorRaw struct {
	Contract *VMCTransactor // Generic write-only contract binding to access the raw methods on
}

// NewVMC creates a new instance of VMC, bound to a specific deployed contract.
func NewVMC(address common.Address, backend bind.ContractBackend) (*VMC, error) {
	contract, err := bindVMC(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &VMC{VMCCaller: VMCCaller{contract: contract}, VMCTransactor: VMCTransactor{contract: contract}}, nil
}

// NewVMCCaller creates a new read-only instance of VMC, bound to a specific deployed contract.
func NewVMCCaller(address common.Address, caller bind.ContractCaller) (*VMCCaller, error) {
	contract, err := bindVMC(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &VMCCaller{contract: contract}, nil
}

// NewVMCTransactor creates a new write-only instance of VMC, bound to a specific deployed contract.
func NewVMCTransactor(address common.Address, transactor bind.ContractTransactor) (*VMCTransactor, error) {
	contract, err := bindVMC(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &VMCTransactor{contract: contract}, nil
}

// bindVMC binds a generic wrapper to an already deployed contract.
func bindVMC(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(VMCABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VMC *VMCRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _VMC.Contract.VMCCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VMC *VMCRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VMC.Contract.VMCTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VMC *VMCRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VMC.Contract.VMCTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VMC *VMCCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _VMC.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VMC *VMCTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VMC.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VMC *VMCTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VMC.Contract.contract.Transact(opts, method, params...)
}

// GetAncestorDistance is a free data retrieval call binding the contract method 0x5badac53.
//
// Solidity: function getAncestorDistance( bytes32) constant returns(bytes32)
func (_VMC *VMCCaller) GetAncestorDistance(opts *bind.CallOpts, arg0 [32]byte) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getAncestorDistance", arg0)
	return *ret0, err
}

// GetAncestorDistance is a free data retrieval call binding the contract method 0x5badac53.
//
// Solidity: function getAncestorDistance( bytes32) constant returns(bytes32)
func (_VMC *VMCSession) GetAncestorDistance(arg0 [32]byte) ([32]byte, error) {
	return _VMC.Contract.GetAncestorDistance(&_VMC.CallOpts, arg0)
}

// GetAncestorDistance is a free data retrieval call binding the contract method 0x5badac53.
//
// Solidity: function getAncestorDistance( bytes32) constant returns(bytes32)
func (_VMC *VMCCallerSession) GetAncestorDistance(arg0 [32]byte) ([32]byte, error) {
	return _VMC.Contract.GetAncestorDistance(&_VMC.CallOpts, arg0)
}

// GetCollationGasLimit is a free data retrieval call binding the contract method 0x934586ec.
//
// Solidity: function getCollationGasLimit() constant returns(uint256)
func (_VMC *VMCCaller) GetCollationGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getCollationGasLimit")
	return *ret0, err
}

// GetCollationGasLimit is a free data retrieval call binding the contract method 0x934586ec.
//
// Solidity: function getCollationGasLimit() constant returns(uint256)
func (_VMC *VMCSession) GetCollationGasLimit() (*big.Int, error) {
	return _VMC.Contract.GetCollationGasLimit(&_VMC.CallOpts)
}

// GetCollationGasLimit is a free data retrieval call binding the contract method 0x934586ec.
//
// Solidity: function getCollationGasLimit() constant returns(uint256)
func (_VMC *VMCCallerSession) GetCollationGasLimit() (*big.Int, error) {
	return _VMC.Contract.GetCollationGasLimit(&_VMC.CallOpts)
}

// GetPeriodStartPrevhash is a free data retrieval call binding the contract method 0x9b33f907.
//
// Solidity: function getPeriodStartPrevhash(_expectedPeriodNumber uint256) constant returns(bytes32)
func (_VMC *VMCCaller) GetPeriodStartPrevhash(opts *bind.CallOpts, _expectedPeriodNumber *big.Int) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getPeriodStartPrevhash", _expectedPeriodNumber)
	return *ret0, err
}

// GetPeriodStartPrevhash is a free data retrieval call binding the contract method 0x9b33f907.
//
// Solidity: function getPeriodStartPrevhash(_expectedPeriodNumber uint256) constant returns(bytes32)
func (_VMC *VMCSession) GetPeriodStartPrevhash(_expectedPeriodNumber *big.Int) ([32]byte, error) {
	return _VMC.Contract.GetPeriodStartPrevhash(&_VMC.CallOpts, _expectedPeriodNumber)
}

// GetPeriodStartPrevhash is a free data retrieval call binding the contract method 0x9b33f907.
//
// Solidity: function getPeriodStartPrevhash(_expectedPeriodNumber uint256) constant returns(bytes32)
func (_VMC *VMCCallerSession) GetPeriodStartPrevhash(_expectedPeriodNumber *big.Int) ([32]byte, error) {
	return _VMC.Contract.GetPeriodStartPrevhash(&_VMC.CallOpts, _expectedPeriodNumber)
}

// GetShardList is a free data retrieval call binding the contract method 0x5e57c86c.
//
// Solidity: function getShardList(_valcodeAddr address) constant returns(bool[100])
func (_VMC *VMCCaller) GetShardList(opts *bind.CallOpts, _valcodeAddr common.Address) ([100]bool, error) {
	var (
		ret0 = new([100]bool)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getShardList", _valcodeAddr)
	return *ret0, err
}

// GetShardList is a free data retrieval call binding the contract method 0x5e57c86c.
//
// Solidity: function getShardList(_valcodeAddr address) constant returns(bool[100])
func (_VMC *VMCSession) GetShardList(_valcodeAddr common.Address) ([100]bool, error) {
	return _VMC.Contract.GetShardList(&_VMC.CallOpts, _valcodeAddr)
}

// GetShardList is a free data retrieval call binding the contract method 0x5e57c86c.
//
// Solidity: function getShardList(_valcodeAddr address) constant returns(bool[100])
func (_VMC *VMCCallerSession) GetShardList(_valcodeAddr common.Address) ([100]bool, error) {
	return _VMC.Contract.GetShardList(&_VMC.CallOpts, _valcodeAddr)
}

// GetValidatorsMaxIndex is a free data retrieval call binding the contract method 0x2b3407f9.
//
// Solidity: function getValidatorsMaxIndex() constant returns(int256)
func (_VMC *VMCCaller) GetValidatorsMaxIndex(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getValidatorsMaxIndex")
	return *ret0, err
}

// GetValidatorsMaxIndex is a free data retrieval call binding the contract method 0x2b3407f9.
//
// Solidity: function getValidatorsMaxIndex() constant returns(int256)
func (_VMC *VMCSession) GetValidatorsMaxIndex() (*big.Int, error) {
	return _VMC.Contract.GetValidatorsMaxIndex(&_VMC.CallOpts)
}

// GetValidatorsMaxIndex is a free data retrieval call binding the contract method 0x2b3407f9.
//
// Solidity: function getValidatorsMaxIndex() constant returns(int256)
func (_VMC *VMCCallerSession) GetValidatorsMaxIndex() (*big.Int, error) {
	return _VMC.Contract.GetValidatorsMaxIndex(&_VMC.CallOpts)
}

// Sample is a free data retrieval call binding the contract method 0xa8c57753.
//
// Solidity: function sample(_shardId int256) constant returns(address)
func (_VMC *VMCCaller) Sample(opts *bind.CallOpts, _shardId *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "sample", _shardId)
	return *ret0, err
}

// Sample is a free data retrieval call binding the contract method 0xa8c57753.
//
// Solidity: function sample(_shardId int256) constant returns(address)
func (_VMC *VMCSession) Sample(_shardId *big.Int) (common.Address, error) {
	return _VMC.Contract.Sample(&_VMC.CallOpts, _shardId)
}

// Sample is a free data retrieval call binding the contract method 0xa8c57753.
//
// Solidity: function sample(_shardId int256) constant returns(address)
func (_VMC *VMCCallerSession) Sample(_shardId *big.Int) (common.Address, error) {
	return _VMC.Contract.Sample(&_VMC.CallOpts, _shardId)
}

// AddHeader is a paid mutator transaction binding the contract method 0xf7fecf7d.
//
// Solidity: function addHeader(_header bytes) returns(bool)
func (_VMC *VMCTransactor) AddHeader(opts *bind.TransactOpts, _header []byte) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "addHeader", _header)
}

// AddHeader is a paid mutator transaction binding the contract method 0xf7fecf7d.
//
// Solidity: function addHeader(_header bytes) returns(bool)
func (_VMC *VMCSession) AddHeader(_header []byte) (*types.Transaction, error) {
	return _VMC.Contract.AddHeader(&_VMC.TransactOpts, _header)
}

// AddHeader is a paid mutator transaction binding the contract method 0xf7fecf7d.
//
// Solidity: function addHeader(_header bytes) returns(bool)
func (_VMC *VMCTransactorSession) AddHeader(_header []byte) (*types.Transaction, error) {
	return _VMC.Contract.AddHeader(&_VMC.TransactOpts, _header)
}

// Deposit is a paid mutator transaction binding the contract method 0xf9609f08.
//
// Solidity: function deposit(_validationCodeAddr address, _returnAddr address) returns(int256)
func (_VMC *VMCTransactor) Deposit(opts *bind.TransactOpts, _validationCodeAddr common.Address, _returnAddr common.Address) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "deposit", _validationCodeAddr, _returnAddr)
}

// Deposit is a paid mutator transaction binding the contract method 0xf9609f08.
//
// Solidity: function deposit(_validationCodeAddr address, _returnAddr address) returns(int256)
func (_VMC *VMCSession) Deposit(_validationCodeAddr common.Address, _returnAddr common.Address) (*types.Transaction, error) {
	return _VMC.Contract.Deposit(&_VMC.TransactOpts, _validationCodeAddr, _returnAddr)
}

// Deposit is a paid mutator transaction binding the contract method 0xf9609f08.
//
// Solidity: function deposit(_validationCodeAddr address, _returnAddr address) returns(int256)
func (_VMC *VMCTransactorSession) Deposit(_validationCodeAddr common.Address, _returnAddr common.Address) (*types.Transaction, error) {
	return _VMC.Contract.Deposit(&_VMC.TransactOpts, _validationCodeAddr, _returnAddr)
}

// TxToShard is a paid mutator transaction binding the contract method 0x372a9e2a.
//
// Solidity: function txToShard(_to address, _shardId int256, _txStartgas uint256, _txGasprice uint256, _data bytes12) returns(int256)
func (_VMC *VMCTransactor) TxToShard(opts *bind.TransactOpts, _to common.Address, _shardId *big.Int, _txStartgas *big.Int, _txGasprice *big.Int, _data [12]byte) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "txToShard", _to, _shardId, _txStartgas, _txGasprice, _data)
}

// TxToShard is a paid mutator transaction binding the contract method 0x372a9e2a.
//
// Solidity: function txToShard(_to address, _shardId int256, _txStartgas uint256, _txGasprice uint256, _data bytes12) returns(int256)
func (_VMC *VMCSession) TxToShard(_to common.Address, _shardId *big.Int, _txStartgas *big.Int, _txGasprice *big.Int, _data [12]byte) (*types.Transaction, error) {
	return _VMC.Contract.TxToShard(&_VMC.TransactOpts, _to, _shardId, _txStartgas, _txGasprice, _data)
}

// TxToShard is a paid mutator transaction binding the contract method 0x372a9e2a.
//
// Solidity: function txToShard(_to address, _shardId int256, _txStartgas uint256, _txGasprice uint256, _data bytes12) returns(int256)
func (_VMC *VMCTransactorSession) TxToShard(_to common.Address, _shardId *big.Int, _txStartgas *big.Int, _txGasprice *big.Int, _data [12]byte) (*types.Transaction, error) {
	return _VMC.Contract.TxToShard(&_VMC.TransactOpts, _to, _shardId, _txStartgas, _txGasprice, _data)
}

// UpdataGasPrice is a paid mutator transaction binding the contract method 0xe551e00a.
//
// Solidity: function updataGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCTransactor) UpdataGasPrice(opts *bind.TransactOpts, _receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "updataGasPrice", _receiptId, _txGasprice)
}

// UpdataGasPrice is a paid mutator transaction binding the contract method 0xe551e00a.
//
// Solidity: function updataGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCSession) UpdataGasPrice(_receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.UpdataGasPrice(&_VMC.TransactOpts, _receiptId, _txGasprice)
}

// UpdataGasPrice is a paid mutator transaction binding the contract method 0xe551e00a.
//
// Solidity: function updataGasPrice(_receiptId int256, _txGasprice uint256) returns(bool)
func (_VMC *VMCTransactorSession) UpdataGasPrice(_receiptId *big.Int, _txGasprice *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.UpdataGasPrice(&_VMC.TransactOpts, _receiptId, _txGasprice)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0e3b2f0e.
//
// Solidity: function withdraw(_validatorIndex int256, _sig bytes32) returns(bool)
func (_VMC *VMCTransactor) Withdraw(opts *bind.TransactOpts, _validatorIndex *big.Int, _sig [32]byte) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "withdraw", _validatorIndex, _sig)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0e3b2f0e.
//
// Solidity: function withdraw(_validatorIndex int256, _sig bytes32) returns(bool)
func (_VMC *VMCSession) Withdraw(_validatorIndex *big.Int, _sig [32]byte) (*types.Transaction, error) {
	return _VMC.Contract.Withdraw(&_VMC.TransactOpts, _validatorIndex, _sig)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0e3b2f0e.
//
// Solidity: function withdraw(_validatorIndex int256, _sig bytes32) returns(bool)
func (_VMC *VMCTransactorSession) Withdraw(_validatorIndex *big.Int, _sig [32]byte) (*types.Transaction, error) {
	return _VMC.Contract.Withdraw(&_VMC.TransactOpts, _validatorIndex, _sig)
}

// ValidatorContractABI is the input ABI used to generate the binding from.
const ValidatorContractABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_sig\",\"type\":\"bytes32\"}],\"name\":\"withdraw\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// ValidatorContractBin is the compiled bytecode used for deploying new contracts.
const ValidatorContractBin = `0x`

// DeployValidatorContract deploys a new Ethereum contract, binding an instance of ValidatorContract to it.
func DeployValidatorContract(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ValidatorContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ValidatorContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ValidatorContractBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ValidatorContract{ValidatorContractCaller: ValidatorContractCaller{contract: contract}, ValidatorContractTransactor: ValidatorContractTransactor{contract: contract}}, nil
}

// ValidatorContract is an auto generated Go binding around an Ethereum contract.
type ValidatorContract struct {
	ValidatorContractCaller     // Read-only binding to the contract
	ValidatorContractTransactor // Write-only binding to the contract
}

// ValidatorContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValidatorContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValidatorContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValidatorContractSession struct {
	Contract     *ValidatorContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ValidatorContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValidatorContractCallerSession struct {
	Contract *ValidatorContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// ValidatorContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValidatorContractTransactorSession struct {
	Contract     *ValidatorContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// ValidatorContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValidatorContractRaw struct {
	Contract *ValidatorContract // Generic contract binding to access the raw methods on
}

// ValidatorContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValidatorContractCallerRaw struct {
	Contract *ValidatorContractCaller // Generic read-only contract binding to access the raw methods on
}

// ValidatorContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValidatorContractTransactorRaw struct {
	Contract *ValidatorContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValidatorContract creates a new instance of ValidatorContract, bound to a specific deployed contract.
func NewValidatorContract(address common.Address, backend bind.ContractBackend) (*ValidatorContract, error) {
	contract, err := bindValidatorContract(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValidatorContract{ValidatorContractCaller: ValidatorContractCaller{contract: contract}, ValidatorContractTransactor: ValidatorContractTransactor{contract: contract}}, nil
}

// NewValidatorContractCaller creates a new read-only instance of ValidatorContract, bound to a specific deployed contract.
func NewValidatorContractCaller(address common.Address, caller bind.ContractCaller) (*ValidatorContractCaller, error) {
	contract, err := bindValidatorContract(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorContractCaller{contract: contract}, nil
}

// NewValidatorContractTransactor creates a new write-only instance of ValidatorContract, bound to a specific deployed contract.
func NewValidatorContractTransactor(address common.Address, transactor bind.ContractTransactor) (*ValidatorContractTransactor, error) {
	contract, err := bindValidatorContract(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &ValidatorContractTransactor{contract: contract}, nil
}

// bindValidatorContract binds a generic wrapper to an already deployed contract.
func bindValidatorContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ValidatorContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidatorContract *ValidatorContractRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ValidatorContract.Contract.ValidatorContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidatorContract *ValidatorContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidatorContract.Contract.ValidatorContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidatorContract *ValidatorContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidatorContract.Contract.ValidatorContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidatorContract *ValidatorContractCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ValidatorContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidatorContract *ValidatorContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidatorContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidatorContract *ValidatorContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidatorContract.Contract.contract.Transact(opts, method, params...)
}

// Withdraw is a paid mutator transaction binding the contract method 0x8e19899e.
//
// Solidity: function withdraw(_sig bytes32) returns(bool)
func (_ValidatorContract *ValidatorContractTransactor) Withdraw(opts *bind.TransactOpts, _sig [32]byte) (*types.Transaction, error) {
	return _ValidatorContract.contract.Transact(opts, "withdraw", _sig)
}

// Withdraw is a paid mutator transaction binding the contract method 0x8e19899e.
//
// Solidity: function withdraw(_sig bytes32) returns(bool)
func (_ValidatorContract *ValidatorContractSession) Withdraw(_sig [32]byte) (*types.Transaction, error) {
	return _ValidatorContract.Contract.Withdraw(&_ValidatorContract.TransactOpts, _sig)
}

// Withdraw is a paid mutator transaction binding the contract method 0x8e19899e.
//
// Solidity: function withdraw(_sig bytes32) returns(bool)
func (_ValidatorContract *ValidatorContractTransactorSession) Withdraw(_sig [32]byte) (*types.Transaction, error) {
	return _ValidatorContract.Contract.Withdraw(&_ValidatorContract.TransactOpts, _sig)
}
