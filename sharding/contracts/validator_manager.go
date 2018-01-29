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
const SigHasherContractBin = `0x60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a72305820c5e12cfe03e1e876e69b3e7102c7e127c4297a29e99b2f672d5d071d1f25699f0029`

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
const VMCABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"getValidatorsMaxIndex\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_shardId\",\"type\":\"int256\"},{\"name\":\"_txStartgas\",\"type\":\"uint256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes12\"}],\"name\":\"txToShard\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"getAncestorDistance\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_validatorAddr\",\"type\":\"address\"}],\"name\":\"getShardList\",\"outputs\":[{\"name\":\"\",\"type\":\"bool[100]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_validatorIndex\",\"type\":\"int256\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getCollationGasLimit\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_expectedPeriodNumber\",\"type\":\"uint256\"}],\"name\":\"getPeriodStartPrevhash\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_shardId\",\"type\":\"int256\"}],\"name\":\"sample\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_receiptId\",\"type\":\"int256\"},{\"name\":\"_txGasprice\",\"type\":\"uint256\"}],\"name\":\"updataGasPrice\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_returnAddr\",\"type\":\"address\"}],\"name\":\"deposit\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_header\",\"type\":\"bytes\"}],\"name\":\"addHeader\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// VMCBin is the compiled bytecode used for deploying new contracts.
const VMCBin = `0x6060604052341561000f57600080fd5b6000600481905560075568056bc75e2d631000006008556005600981905562061a80600a55600c556064600d819055600e556040517f6164645f686561646572282900000000000000000000000000000000000000008152600c01604051908190039020600f5560108054600160a060020a03191673dffd41e18f04ad8810c83b14fd1426a82e625a7d179055611340806100ab6000396000f3006060604052600436106100ae5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416632b3407f981146100b3578063372a9e2a146100d85780635badac53146101025780635e57c86c146101185780637e62eab814610170578063934586ec146101885780639b33f9071461019b578063a8c57753146101b1578063e551e00a146101e3578063f340fa0114610205578063f7fecf7d14610219575b600080fd5b34156100be57600080fd5b6100c661026a565b60405190815260200160405180910390f35b6100c6600160a060020a0360043516602435604435606435600160a060020a0319608435166102df565b341561010d57600080fd5b6100c660043561041b565b341561012357600080fd5b610137600160a060020a0360043516610421565b6040518082610c8080838360005b8381101561015d578082015183820152602001610145565b5050505090500191505060405180910390f35b341561017b57600080fd5b61018660043561055c565b005b341561019357600080fd5b6100c6610689565b34156101a657600080fd5b6100c6600435610691565b34156101bc57600080fd5b6101c76004356106b5565b604051600160a060020a03909116815260200160405180910390f35b6101f16004356024356107cc565b604051901515815260200160405180910390f35b6100c6600160a060020a0360043516610811565b341561022457600080fd5b6101f160046024813581810190830135806020601f8201819004810201604051908101604052818152929190602084018383808284375094965061099b95505050505050565b60008060008060008093506009544381151561028257fe5b059250600754600454019150600090505b6104008112156102d1578181126102a9576102d1565b6000818152602081905260409020600301548390136102c9576001840193505b600101610293565b600754840194505050505090565b60008060e060405190810160409081528782526020808301889052818301879052346060840152600160a060020a0333811660808501528a1660a0840152600160a060020a0319861660c08401526005546000908152600290915220815181556020820151816001015560408201518160020155606082015181600301556080820151600482018054600160a060020a031916600160a060020a039290921691909117905560a0820151600582018054600160a060020a031916600160a060020a039290921691909117905560c0820151600690910155505060058054600181019091558086600160a060020a0389166040517f74785f746f5f73686172642829000000000000000000000000000000000000008152600d01604051809103902060405190815260200160405180910390a39695505050505050565b50600090565b610429611246565b610431611246565b60008060008060008060006009544381151561044957fe5b05965060016009548802039550600086121561046457600095505b8540945061047061026a565b6004549094501561054e57600092505b60648360ff16101561054e5760008860ff85166064811061049d57fe5b91151560209092020152600091505b60648260ff16101561054357838560ff8086169085166040519283526020830191909152604080830191909152606090910190519081900390208115156104ef57fe5b06600081815260208190526040902060010154909150600160a060020a038b8116911614156105385760018860ff85166064811061052957fe5b91151560209092020152610543565b8160010191506104ac565b826001019250610480565b509598975050505050505050565b60006040517f7769746864726177000000000000000000000000000000000000000000000000815260080160405190819003902060008381526020819052604090206001015490915033600160a060020a039081169116146105bd57600080fd5b6000828152602081905260409081902060028101549054600160a060020a039091169181156108fc02919051600060405180830381858888f19350505050151561060657600080fd5b600082815260208181526040808320600181018054600160a060020a03168552600b8452918420805460ff19169055858452918390528282558054600160a060020a031990811690915560028201805490911690556003015561066882610d36565b60048054600019019055818160405190815260200160405180910390a15050565b629896805b90565b600c54600090820260001901438190116106aa57600080fd5b804091505b50919050565b6000806000806000806000600c5443101515156106d157600080fd5b600954438115156106de57fe5b0595506001600954870203945060008512156106f957600094505b600c54854094506001904381151561070d57fe5b0643030340600190049250600d54838960010260405191825260208201526040908101905190819003902081151561074157fe5b06915061074c61026a565b84898460405192835260208301919091526040808301919091526060909101905190819003902081151561077c57fe5b06600081815260208190526040902060030154909150869013156107a357600096506107c1565b600081815260208190526040902060010154600160a060020a031696505b505050505050919050565b60008281526002602052604081206004015433600160a060020a039081169116146107f657600080fd5b50600091825260026020819052604090922090910155600190565b600160a060020a0333166000908152600b60205260408120548190819060ff161561083b57600080fd5b600854341461084957600080fd5b506000610854610d55565b151561086957610862610d5c565b915061091c565b60045491506009544381151561087b57fe5b05600101905060806040519081016040908152348252600160a060020a03338116602080850191909152908716828401526060830184905260008581529081905220815181556020820151600182018054600160a060020a031916600160a060020a03929092169190911790556040820151600282018054600160a060020a031916600160a060020a03929092169190911790556060820151600390910155505b600480546001908101909155600160a060020a0333166000818152600b602052604090819020805460ff19169093179092558391517f6465706f736974282900000000000000000000000000000000000000000000008152600901604051809103902060405190815260200160405180910390a28192505b5050919050565b60006109a561126f565b6109ad611281565b6109b56112a2565b84925060008080806109d66109d188600163ffffffff610d9316565b610df7565b9550610140604051908101604052806109f66109f189610e30565b610e72565b8152602001610a0c610a0789610e30565b610e83565b8152602001610a1d6109f189610e30565b8152602001610a2e6109f189610e30565b8152602001610a3f6109f189610e30565b8152602001610a55610a5089610e30565b610ed4565b600160a060020a03168152602001610a6f6109f189610e30565b8152602001610a806109f189610e30565b8152602001610a916109f189610e30565b8152602001610aa7610aa289610e30565b610f1e565b905294506000855112158015610abf5750600e548551125b1515610aca57600080fd5b600c54431015610ad957600080fd5b600c5443811515610ae657fe5b04856020015114610af657600080fd5b6001600c548660200151020340604086015114610b1257600080fd5b846040519081526020016040519081900390209350831515610b3057fe5b60016000865181526020808201929092526040908101600090812087825290925290206001015415610b5e57fe5b846060015115610bae5784606001511580610ba65750600060018187518152602001908152602001600020600087606001518152602081019190915260400160002060010154135b1515610bae57fe5b8460200151601160008751815260200190815260200160002054121515610bd157fe5b610bdb85516106b5565b9250600160a060020a0383161515610bf65760009750610d2a565b600160008651815260200190815260200160002060008660600151815260208101919091526040016000206001908101540191508161010086015114610c3857fe5b6040805190810160405280866060015181526020018390526001600087518152602080820192909252604090810160009081208882529092529020815181556020820151600190910155506020850151601160008751815260200190815260200160002081905550600160008660000151815260200190815260200160002060006003600088600001518152602001908152602001600020546000191660001916815260200190815260200160002060010154821315610d255760036000865181526020019081526020016000205490508360036000876000015181526020810191909152604001600020555b600197505b50505050505050919050565b6007805460009081526006602052604090209190915580546001019055565b6007541590565b6000610d66610d55565b15610d74575060001961068e565b5060078054600019019081905560009081526006602052604090205490565b610d9b6112fd565b610da36112fd565b6000610dae85610f6c565b91508315610def578451905080610dc483610fbd565b1115610dcc57fe5b80610dd7835161103c565b14610dde57fe5b610de7826110ce565b1515610def57fe5b509392505050565b610dff611281565b6000610e0a83611110565b1515610e1557600080fd5b610e1e83610fbd565b83519383529092016020820152919050565b610e386112fd565b600080610e448461113b565b156100ae5783602001519150610e598261103c565b8284526020808501829052838201908601529050610994565b6000610e7d82610e83565b92915050565b6000806000610e918461115e565b1515610e9c57600080fd5b610ea584611188565b9150915060208111158015610eb957508015155b1515610ec157fe5b806020036101000a825104949350505050565b6000806000610ee28461115e565b1515610eed57600080fd5b610ef684611188565b909250905060148114610f0557fe5b6c01000000000000000000000000825104949350505050565b610f2661126f565b600082602001519050801515610f3b576106af565b80604051805910610f495750595b818152601f19601f8301168101602001604052905091506106af83518383611205565b610f746112fd565b60008083519150811515610f9d5760408051908101604052600080825260208201529250610994565b506020830160408051908101604052908152602081019190915292915050565b600080600083602001511515610fd65760009250610994565b83519050805160001a91506080821015610ff35760009250610994565b60b882108061100e575060c0821015801561100e575060f882105b1561101c5760019250610994565b60c08210156110315760b51982019250610994565b5060f5190192915050565b600080825160001a9050608081101561105857600191506106af565b60b881101561106d57607e19810191506106af565b60c08110156110975760b78103806020036101000a600185015104808201600101935050506106af565b60f88110156110ac5760be19810191506106af565b60f78103806020036101000a6001850151048082016001019350505050919050565b600080808084519050805160001a9250805160011a91506081831480156110f55750608082105b156111035760009350611108565b600193505b505050919050565b6000808260200151151561112757600091506106af565b8251905060c0815160001a10159392505050565b60006111456112fd565b8251905080602001518151018360200151109392505050565b6000808260200151151561117557600091506106af565b8251905060c0815160001a109392505050565b60008060008060006111998661115e565b15156111a457600080fd5b85519150815160001a925060808310156111c457819450600193506111fd565b60b88310156111e257600186602001510393508160010194506111fd565b5060b619820180600160208801510303935080820160010194505b505050915091565b60006020601f83010484602085015b8284146112335760208402808301518183015260018501945050611214565b6000865160200187015250505050505050565b610c806040519081016040526064815b6000815260001990910190602001816112565790505090565b60206040519081016040526000815290565b6060604051908101604052806112956112fd565b8152602001600081525090565b6101406040519081016040908152600080835260208301819052908201819052606082018190526080820181905260a0820181905260c0820181905260e0820181905261010082015261012081016112f861126f565b905290565b6040805190810160405260008082526020820152905600a165627a7a723058201e5dfd4590fe444b86534a5417adf73abd6edb65742c9c70d8223afa491d76730029`

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
// Solidity: function getShardList(_validatorAddr address) constant returns(bool[100])
func (_VMC *VMCCaller) GetShardList(opts *bind.CallOpts, _validatorAddr common.Address) ([100]bool, error) {
	var (
		ret0 = new([100]bool)
	)
	out := ret0
	err := _VMC.contract.Call(opts, out, "getShardList", _validatorAddr)
	return *ret0, err
}

// GetShardList is a free data retrieval call binding the contract method 0x5e57c86c.
//
// Solidity: function getShardList(_validatorAddr address) constant returns(bool[100])
func (_VMC *VMCSession) GetShardList(_validatorAddr common.Address) ([100]bool, error) {
	return _VMC.Contract.GetShardList(&_VMC.CallOpts, _validatorAddr)
}

// GetShardList is a free data retrieval call binding the contract method 0x5e57c86c.
//
// Solidity: function getShardList(_validatorAddr address) constant returns(bool[100])
func (_VMC *VMCCallerSession) GetShardList(_validatorAddr common.Address) ([100]bool, error) {
	return _VMC.Contract.GetShardList(&_VMC.CallOpts, _validatorAddr)
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

// Deposit is a paid mutator transaction binding the contract method 0xf340fa01.
//
// Solidity: function deposit(_returnAddr address) returns(int256)
func (_VMC *VMCTransactor) Deposit(opts *bind.TransactOpts, _returnAddr common.Address) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "deposit", _returnAddr)
}

// Deposit is a paid mutator transaction binding the contract method 0xf340fa01.
//
// Solidity: function deposit(_returnAddr address) returns(int256)
func (_VMC *VMCSession) Deposit(_returnAddr common.Address) (*types.Transaction, error) {
	return _VMC.Contract.Deposit(&_VMC.TransactOpts, _returnAddr)
}

// Deposit is a paid mutator transaction binding the contract method 0xf340fa01.
//
// Solidity: function deposit(_returnAddr address) returns(int256)
func (_VMC *VMCTransactorSession) Deposit(_returnAddr common.Address) (*types.Transaction, error) {
	return _VMC.Contract.Deposit(&_VMC.TransactOpts, _returnAddr)
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

// Withdraw is a paid mutator transaction binding the contract method 0x7e62eab8.
//
// Solidity: function withdraw(_validatorIndex int256) returns()
func (_VMC *VMCTransactor) Withdraw(opts *bind.TransactOpts, _validatorIndex *big.Int) (*types.Transaction, error) {
	return _VMC.contract.Transact(opts, "withdraw", _validatorIndex)
}

// Withdraw is a paid mutator transaction binding the contract method 0x7e62eab8.
//
// Solidity: function withdraw(_validatorIndex int256) returns()
func (_VMC *VMCSession) Withdraw(_validatorIndex *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.Withdraw(&_VMC.TransactOpts, _validatorIndex)
}

// Withdraw is a paid mutator transaction binding the contract method 0x7e62eab8.
//
// Solidity: function withdraw(_validatorIndex int256) returns()
func (_VMC *VMCTransactorSession) Withdraw(_validatorIndex *big.Int) (*types.Transaction, error) {
	return _VMC.Contract.Withdraw(&_VMC.TransactOpts, _validatorIndex)
}
