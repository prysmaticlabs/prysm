package eth1

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

// NetworkId is the ID of the ETH1 chain.
const NetworkId = 1337

// KeystorePassword is the password used to decrypt ETH1 keystores.
const KeystorePassword = "password"

const minerPasswordFile = "password.txt"
const minerFile = "UTC--2021-12-22T19-14-08.590377700Z--878705ba3f8bc32fcf7f4caa1a35e72af65cf766"
const timeGapPerTX = 100 * time.Millisecond
const staticFilesPath = "/testing/endtoend/static-files/eth1"
const timeGapPerMiningTX = 250 * time.Millisecond

var _ e2etypes.ComponentRunner = (*NodeSet)(nil)
var _ e2etypes.MultipleComponentRunners = (*NodeSet)(nil)
var _ e2etypes.MultipleComponentRunners = (*ProxySet)(nil)
var _ e2etypes.ComponentRunner = (*Miner)(nil)
var _ e2etypes.ComponentRunner = (*Node)(nil)
var _ e2etypes.EngineProxy = (*Proxy)(nil)

// WaitForBlocks waits for a certain amount of blocks to be mined by the ETH1 chain before returning.
func WaitForBlocks(web3 *ethclient.Client, keystore *keystore.Key, blocksToWait uint64) error {
	nonce, err := web3.PendingNonceAt(context.Background(), keystore.Address)
	if err != nil {
		return err
	}
	chainID, err := web3.NetworkID(context.Background())
	if err != nil {
		return err
	}
	block, err := web3.BlockByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	finishBlock := block.NumberU64() + blocksToWait

	for block.NumberU64() <= finishBlock {
		spamTX := types.NewTransaction(nonce, keystore.Address, big.NewInt(0), 21000, big.NewInt(1e6), []byte{})
		signed, err := types.SignTx(spamTX, types.NewEIP155Signer(chainID), keystore.PrivateKey)
		if err != nil {
			return err
		}
		if err = web3.SendTransaction(context.Background(), signed); err != nil {
			return err
		}
		nonce++
		time.Sleep(timeGapPerMiningTX)
		block, err = web3.BlockByNumber(context.Background(), nil)
		if err != nil {
			return err
		}
	}
	return nil
}
