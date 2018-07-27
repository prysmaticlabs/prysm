package attester

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prysmaticlabs/prysm/client/contracts"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	shardparams "github.com/prysmaticlabs/prysm/client/params"
	"github.com/sirupsen/logrus"
)

// subscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible attester for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the SMC to create a collation.
func subscribeBlockHeaders(reader mainchain.Reader, caller mainchain.ContractCaller, account *accounts.Account) error {
	headerChan := make(chan *gethTypes.Header, 16)

	_, err := reader.SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		return fmt.Errorf("unable to subscribe to incoming headers. %v", err)
	}

	log.Info("Listening for new headers...")

	for {
		// TODO: Error handling for getting disconnected from the client.
		head := <-headerChan
		// Query the current state to see if we are an eligible attester.
		log.WithFields(logrus.Fields{
			"number": head.Number.String(),
		}).Info("Received new header")

		// Check if we are in the attester pool before checking if we are an eligible attester.
		v, err := isAccountInAttesterPool(caller, account)
		if err != nil {
			return fmt.Errorf("unable to verify client in attester pool. %v", err)
		}

		if v {
			if err := checkSMCForAttester(caller, account); err != nil {
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		}
	}
}

// checkSMCForAttester checks if we are an eligible attester for
// collation for the available shards in the SMC. The function calls
// getEligibleAttester from the SMC and attester a collation if
// conditions are met.
func checkSMCForAttester(caller mainchain.ContractCaller, account *accounts.Account) error {
	log.Info("Checking if we are an eligible collation attester for a shard...")
	shardCount, err := caller.GetShardCount()
	if err != nil {
		return fmt.Errorf("can't get shard count from smc: %v", err)
	}
	for s := int64(0); s < shardCount; s++ {
		// Checks if we are an eligible attester according to the SMC.
		addr, err := caller.SMCCaller().GetAttesterInCommittee(&bind.CallOpts{}, big.NewInt(s))

		if err != nil {
			return err
		}

		if addr == account.Address {
			log.Infof("Selected as attester on shard: %d", s)
		}

	}

	return nil
}

// getAttesterRegistry retrieves the registry of the registered account.
func getAttesterRegistry(caller mainchain.ContractCaller, account *accounts.Account) (*contracts.Registry, error) {

	var nreg contracts.Registry
	nreg, err := caller.SMCCaller().AttesterRegistry(&bind.CallOpts{}, account.Address)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve attester registry: %v", err)
	}

	return &nreg, nil
}

// isAccountInAttesterPool checks if the user is in the attester pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsAttesterDeposited from the SMC and returns true if
// the user is in the attester pool.
func isAccountInAttesterPool(caller mainchain.ContractCaller, account *accounts.Account) (bool, error) {

	nreg, err := getAttesterRegistry(caller, account)
	if err != nil {
		return false, err
	}

	if !nreg.Deposited {
		log.Warnf("Account %s not in attester pool.", account.Address.Hex())
	}

	return nreg.Deposited, nil
}

// joinAttesterPool checks if the deposit flag is true and the account is a
// attester in the SMC. If the account is not in the set, it will deposit ETH
// into contract.
func joinAttesterPool(manager mainchain.ContractManager, client mainchain.EthClient) error {
	if !client.DepositFlag() {
		return errors.New("joinAttesterPool called when deposit flag was not set")
	}

	if b, err := isAccountInAttesterPool(manager, client.Account()); b || err != nil {
		if b {
			log.Info("Already joined attester pool")
			return nil
		}
		return err
	}

	log.Info("Joining attester pool")
	deposit := shardparams.DefaultAttesterDeposit()
	txOps, err := manager.CreateTXOpts(deposit)
	if err != nil {
		return fmt.Errorf("unable to initiate the deposit transaction: %v", err)
	}

	tx, err := manager.SMCTransactor().RegisterAttester(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a attester: %v", err)
	}

	err = client.WaitForTransaction(context.Background(), tx.Hash(), 400)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status == gethTypes.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to deposit ETH and become a attester")
	}

	if inPool, err := isAccountInAttesterPool(manager, client.Account()); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in attester pool")
	}

	log.Infof("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(deposit, big.NewInt(params.Ether)), tx.Hash().Hex())

	return nil
}
