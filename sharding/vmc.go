package sharding

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// TODO: Remove bytecode variable, reference ABI from solidity, compile solidity file.
var (
	// ABI/bytecode from https://github.com/enriquefynn/go-ethereum/blob/b1475e7c233d42d5c28d12826f8ee03b25cce8ae/sharding/contracts/validator_manager.sol
	abi         = `[{"constant":true,"inputs":[],"name":"getValidatorsMaxIndex","outputs":[{"name":"","type":"int256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_shardId","type":"int256"},{"name":"_txStartgas","type":"uint256"},{"name":"_txGasprice","type":"uint256"},{"name":"_data","type":"bytes12"}],"name":"txToShard","outputs":[{"name":"","type":"int256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"getAncestorDistance","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"pure","type":"function"},{"constant":false,"inputs":[{"name":"header","type":"bytes12"}],"name":"addHeader","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_valcodeAddr","type":"address"}],"name":"getShardList","outputs":[{"name":"","type":"bool[100]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getCollationGasLimit","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"pure","type":"function"},{"constant":true,"inputs":[{"name":"_expectedPeriodNumber","type":"uint256"}],"name":"getPeriodStartPrevhash","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_shardId","type":"int256"}],"name":"sample","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_validatorIndex","type":"uint256"},{"name":"_sig","type":"bytes10"}],"name":"withdraw","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_receiptId","type":"int256"},{"name":"_txGasprice","type":"uint256"}],"name":"updataGasPrice","outputs":[{"name":"","type":"bool"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"_validationCodeAddr","type":"address"},{"name":"_returnAddr","type":"address"}],"name":"deposit","outputs":[{"name":"","type":"int256"}],"payable":true,"stateMutability":"payable","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`
	abiBytecode = common.Hex2Bytes("6060604052341561000f57600080fd5b6000600481905550600060078190555068056bc75e2d63100000600881905550600560098190555062061a80600a819055506005600c819055506064600d819055506064600e8190555060405180807f6164645f68656164657228290000000000000000000000000000000000000000815250600c0190506040518091039020600f816000191690555073dffd41e18f04ad8810c83b14fd1426a82e625a7d601060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610e8f806100fd6000396000f3006060604052600436106100af576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680632b3407f9146100b4578063372a9e2a146100dd5780635badac531461015a5780635d106b1f1461019d5780635e57c86c146101ef578063934586ec146102645780639b33f9071461028d578063a8c57753146102cc578063d2db8be11461032f578063e551e00a1461038c578063f9609f08146103c5575b600080fd5b34156100bf57600080fd5b6100c7610426565b6040518082815260200191505060405180910390f35b610144600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190803590602001909190803590602001909190803573ffffffffffffffffffffffffffffffffffffffff1916906020019091905050610519565b6040518082815260200191505060405180910390f35b341561016557600080fd5b61017f600480803560001916906020019091905050610745565b60405180826000191660001916815260200191505060405180910390f35b34156101a857600080fd5b6101d5600480803573ffffffffffffffffffffffffffffffffffffffff191690602001909190505061074c565b604051808215151515815260200191505060405180910390f35b34156101fa57600080fd5b610226600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610753565b6040518082606460200280838360005b83811015610251578082015181840152602081019050610236565b5050505090500191505060405180910390f35b341561026f57600080fd5b610277610900565b6040518082815260200191505060405180910390f35b341561029857600080fd5b6102ae600480803590602001909190505061090b565b60405180826000191660001916815260200191505060405180910390f35b34156102d757600080fd5b6102ed6004808035906020019091905050610930565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561033a57600080fd5b610372600480803590602001909190803575ffffffffffffffffffffffffffffffffffffffffffff1916906020019091905050610a90565b604051808215151515815260200191505060405180910390f35b6103ab6004808035906020019091908035906020019091905050610a98565b604051808215151515815260200191505060405180910390f35b610410600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610b2f565b6040518082815260200191505060405180910390f35b60008060008060008060009450600093506009544381151561044457fe5b059250600754600454019150600090505b61040081121561050a57818112151561046d5761050a565b8473ffffffffffffffffffffffffffffffffffffffff1660008083815260200190815260200160002060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141580156104f35750826000808381526020019081526020016000206003015413155b156104ff576001840193505b806001019050610455565b60075484019550505050505090565b60008060e0604051908101604052808781526020018681526020018581526020013481526020013373ffffffffffffffffffffffffffffffffffffffff1681526020018873ffffffffffffffffffffffffffffffffffffffff1681526020018473ffffffffffffffffffffffffffffffffffffffff19168152506002600060055481526020019081526020016000206000820151816000015560208201518160010155604082015181600201556060820151816003015560808201518160040160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060a08201518160050160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060c08201518160050160146101000a8154816bffffffffffffffffffffffff02191690837401000000000000000000000000000000000000000090040217905550905050600554905060056000815460010191905081905550806001026000191686600102600019168873ffffffffffffffffffffffffffffffffffffffff166001026000191660405180807f74785f746f5f7368617264282900000000000000000000000000000000000000815250600d019050604051809103902060405180826000191660001916815260200191505060405180910390a38091505095945050505050565b6000919050565b6000919050565b61075b610e38565b610763610e38565b60008060008060008060006009544381151561077b57fe5b05965060016009548802039550600086121561079657600095505b854094506107a2610426565b935060006004541415156108f057600092505b60648360ff1610156108ef576000888460ff166064811015156107d457fe5b602002019015159081151581525050600091505b60648212156108e45783858460ff166001028460010260405180846000191660001916815260200183600019166000191681526020018260001916600019168152602001935050505060405180910390206001900481151561084657fe5b06905060008082815260200190815260200160002060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168a73ffffffffffffffffffffffffffffffffffffffff1614156108d9576001888460ff166064811015156108c557fe5b6020020190151590811515815250506108e4565b8160010191506107e8565b8260010192506107b5565b5b8798505050505050505050919050565b600062989680905090565b6000806001600c548402039050804311151561092657600080fd5b8040915050919050565b6000806000806000806000600c54431015151561094c57600080fd5b6009544381151561095957fe5b05955060016009548702039450600085121561097457600094505b84406001900493506001600c544381151561098b57fe5b0643030340600190049250600d548389600102604051808381526020018260001916600019168152602001925050506040518091039020600190048115156109cf57fe5b0691506109da610426565b84896001028460010260405180848152602001836000191660001916815260200182600019166000191681526020019350505050604051809103902060019004811515610a2357fe5b06905085600080838152602001908152602001600020600301541315610a4c5760009650610a85565b60008082815260200190815260200160002060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1696505b505050505050919050565b600092915050565b60003373ffffffffffffffffffffffffffffffffffffffff166002600085815260200190815260200160002060040160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141515610b0a57600080fd5b8160026000858152602001908152602001600020600201819055506001905092915050565b6000806000600b60008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff16151515610b8d57600080fd5b60085434141515610b9d57600080fd5b60009050610ba9610dc8565b1515610bbe57610bb7610dd4565b9150610cdc565b6004549150600160095443811515610bd257fe5b050190506080604051908101604052803481526020018673ffffffffffffffffffffffffffffffffffffffff1681526020018573ffffffffffffffffffffffffffffffffffffffff168152602001828152506000808481526020019081526020016000206000820151816000015560208201518160010160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060408201518160020160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550606082015181600301559050505b600460008154600101919050819055506001600b60008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff02191690831515021790555081600102600019168573ffffffffffffffffffffffffffffffffffffffff166001026000191660405180807f6465706f736974282900000000000000000000000000000000000000000000008152506009019050604051809103902060405180826000191660001916815260200191505060405180910390a2819250505092915050565b60008060075414905090565b6000610dde610dc8565b15610e0b577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9050610e35565b60076000815460019003919050819055506006600060075481526020019081526020016000205490505b90565b610c80604051908101604052806064905b60001515815260200190600190039081610e4957905050905600a165627a7a7230582012b9de3cdd06b3fb95382c1edb67cb60fea466dc693be0c728d39a4b8b78ac840029")
)

// Verify validator management contract.
// Checks that the contract exists and verifies the bytecode. Otherwise, deploys a copy of the contract.
func (c *Client) verifyVMC() error {
	// TODO: Fetch validator manager contract.
	b, err := c.client.CodeAt(context.Background(), validatorManagerAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to get contract code at %s. %v", validatorManagerAddress, err)
	}

	if len(b) == 0 {
		log.Info(fmt.Sprintf("No validator management contract found at %s.", validatorManagerAddress.String()))

		addr, err := c.deployVMC()
		if err != nil {
			return fmt.Errorf("failed to deploy validator management contract: %v", err)
		}

		log.Info(fmt.Sprintf("Created contract at address %s", addr.String()))

		if err != nil {
			return fmt.Errorf("could not deploy validator management contract: %v", err)
		}
	} else {
		// TODO: Check contract bytecode is what we expected, otherwise return error.
	}

	return nil
}

func (c *Client) deployVMC() (*common.Address, error) {
	accounts := c.keystore.Accounts()
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts found")
	}

	// TODO: call unlock only if account is actually locked.
	if err := c.unlockAccount(accounts[0]); err != nil {
		return nil, fmt.Errorf("failed to unlock account 0: %v", err)
	}

	suggestedGasPrice, err := c.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get suggested gas price from node: %v", err)
	}

	nonce, err := c.client.NonceAt(context.Background(), accounts[0].Address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce for %s: %v", accounts[0].Address, err)
	}

	tx := types.NewContractCreation(nonce, new(big.Int).SetInt64(0), contractGasLimit, suggestedGasPrice, abiBytecode)
	signed, err := c.keystore.SignTx(accounts[0], tx, new(big.Int).SetInt64(1000))
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	log.Info(fmt.Sprintf("Creating validator management contract. Tx: %s", signed.Hash().String()))

	if err := c.client.SendTransaction(context.Background(), signed); err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}

	log.Info(fmt.Sprintf("Contract creation sent. Waiting for transaction receipt."))

	for pending := true; pending; _, pending, err = c.client.TransactionByHash(context.Background(), signed.Hash()) {
		if err != nil {
			return nil, fmt.Errorf("Failed to get transaction by hash: %v", err)
		}
		time.Sleep(1 * time.Second)
	}

	receipt, err := c.client.TransactionReceipt(context.Background(), signed.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	return &receipt.ContractAddress, nil
}

func (c *Client) unlockAccount(account accounts.Account) error {
	pass := ""

	if c.ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		blob, err := ioutil.ReadFile(c.ctx.GlobalString(utils.PasswordFileFlag.Name))
		if err != nil {
			return fmt.Errorf("failed to read account password contents in file %s. %v", utils.PasswordFileFlag.Value, err)
		}
		// Some text files end in new line, remove with strings.Trim.
		pass = strings.Trim(string(blob), "\n")
	}

	return c.keystore.Unlock(account, pass)
}
