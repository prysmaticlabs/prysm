package notary

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
	log "github.com/sirupsen/logrus"
)

// subscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible notary for collations. Then, it finds the pending tx's
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
		// Query the current state to see if we are an eligible notary.
		log.Infof("Received new header: %v", head.Number.String())

		// Check if we are in the notary pool before checking if we are an eligible notary.
		v, err := isAccountInNotaryPool(caller, account)
		if err != nil {
			return fmt.Errorf("unable to verify client in notary pool. %v", err)
		}

		if v {
			if err := checkSMCForNotary(caller, account); err != nil {
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		}
	}
}

// checkSMCForNotary checks if we are an eligible notary for
// collation for the available shards in the SMC. The function calls
// getEligibleNotary from the SMC and notary a collation if
// conditions are met.
func checkSMCForNotary(caller mainchain.ContractCaller, account *accounts.Account) error {
	log.Info("Checking if we are an eligible collation notary for a shard...")
	shardCount, err := caller.GetShardCount()
	if err != nil {
		return fmt.Errorf("can't get shard count from smc: %v", err)
	}
	for s := int64(0); s < shardCount; s++ {
		// Checks if we are an eligible notary according to the SMC.
		addr, err := caller.SMCCaller().GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(s))

		if err != nil {
			return err
		}

		if addr == account.Address {
			log.Infof("Selected as notary on shard: %d", s)
		}

	}

	return nil
}

// getNotaryRegistry retrieves the registry of the registered account.
func getNotaryRegistry(caller mainchain.ContractCaller, account *accounts.Account) (*contracts.Registry, error) {

	var nreg contracts.Registry
	nreg, err := caller.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve notary registry: %v", err)
	}

	return &nreg, nil
}

// isAccountInNotaryPool checks if the user is in the notary pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsNotaryDeposited from the SMC and returns true if
// the user is in the notary pool.
func isAccountInNotaryPool(caller mainchain.ContractCaller, account *accounts.Account) (bool, error) {

	nreg, err := getNotaryRegistry(caller, account)
	if err != nil {
		return false, err
	}

	if !nreg.Deposited {
		log.Warnf("Account %s not in notary pool.", account.Address.Hex())
	}

	return nreg.Deposited, nil
}

// joinNotaryPool checks if the deposit flag is true and the account is a
// notary in the SMC. If the account is not in the set, it will deposit ETH
// into contract.
func joinNotaryPool(manager mainchain.ContractManager, client mainchain.EthClient) error {
	if !client.DepositFlag() {
		return errors.New("joinNotaryPool called when deposit flag was not set")
	}

	if b, err := isAccountInNotaryPool(manager, client.Account()); b || err != nil {
		if b {
			log.Info("Already joined notary pool")
			return nil
		}
		return err
	}

	log.Info("Joining notary pool")
	txOps, err := manager.CreateTXOpts(shardparams.DefaultConfig.NotaryDeposit)
	if err != nil {
		return fmt.Errorf("unable to initiate the deposit transaction: %v", err)
	}

	tx, err := manager.SMCTransactor().RegisterNotary(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a notary: %v", err)
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
		return errors.New("transaction was not successful, unable to deposit ETH and become a notary")
	}

	if inPool, err := isAccountInNotaryPool(manager, client.Account()); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in notary pool")
	}

	log.Infof("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(shardparams.DefaultConfig.NotaryDeposit, big.NewInt(params.Ether)), tx.Hash().Hex())

	return nil
}
