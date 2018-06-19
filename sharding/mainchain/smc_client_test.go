package mainchain

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

var (
	key, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr                     = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance1001Eth, _ = new(big.Int).SetString("1001000000000000000000", 10)
)

// Verifies that SMCCLient implements the sharding Service inteface.
var _ = sharding.Service(&SMCClient{})

// fakeClient is struct to implement the smcClient methods for testing.
type fakeClient struct {
	smc         *contracts.SMC
	depositFlag bool
	t           *testing.T
	backend     *backends.SimulatedBackend
}

// Mirrors the function in the main file, but instead of having a client to perform rpc calls
// it is replaced by the simulated backend.
func (f *fakeClient) WaitForTransaction(ctx context.Context, hash common.Hash, durationInSeconds int64) error {

	var receipt *types.Receipt

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(durationInSeconds)*time.Second)

	for err := error(nil); receipt == nil; receipt, err = f.backend.TransactionReceipt(ctxTimeout, hash) {

		if err != nil {
			cancel()
			return fmt.Errorf("unable to retrieve transaction: %v", err)
		}
		if ctxTimeout.Err() != nil {
			cancel()
			return fmt.Errorf("transaction timed out, transaction was not able to be mined in the duration: %v", ctxTimeout.Err())
		}
	}
	cancel()
	ctxTimeout.Done()
	log.Info(fmt.Sprintf("Transaction: %s has been mined", hash.Hex()))
	return nil
}

// Creates and send Fake Transactions to the backend to be mined, takes in the context and
// the current blocknumber as an argument and returns the signed transaction after it has been sent.
func (f *fakeClient) CreateAndSendFakeTx(ctx context.Context, blocknumber int64) (*types.Transaction, error) {
	tx := types.NewTransaction(uint64(blocknumber), common.HexToAddress("0x"), nil, 50000, nil, nil)
	signedtx, err := types.SignTx(tx, types.MakeSigner(&params.ChainConfig{}, big.NewInt(blocknumber)), key)
	if err != nil {
		return nil, err
	}
	err = f.backend.SendTransaction(ctx, signedtx)
	if err != nil {
		return nil, fmt.Errorf("unable to send transaction: %v", err)
	}
	return signedtx, nil
}

func setup() *backends.SimulatedBackend {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	return backend
}

// TestWaitForTransaction tests the WaitForTransaction function in the smcClient, however since for testing we do not have
// an inmemory rpc server to interact with the simulated backend, we have to define the function in the test rather than
// in `smc_client.go`.
// TODO: Test the function in the main file instead of defining it here if a rpc server for testing can be
// implemented.
func TestWaitForTransaction(t *testing.T) {
	backend := setup()
	client := &fakeClient{backend: backend}
	ctx := context.Background()
	timeout := int64(5)

	tx, err := client.CreateAndSendFakeTx(ctx, 0)
	if err != nil {
		t.Error(err)
	}

	receipt, err := client.backend.TransactionReceipt(ctx, tx.Hash())
	if receipt != nil {
		t.Errorf("transaction mined despite backend not being commited: %v", receipt)
	}
	err = client.WaitForTransaction(ctx, tx.Hash(), timeout)
	if err == nil {
		t.Error("transaction is supposed to timeout and return a error")
	}

	// Tests transaction timing out when the block is mined immediately
	// in the timeout period
	go func() {
		newErr := client.WaitForTransaction(ctx, tx.Hash(), timeout)
		if newErr != nil {
			t.Errorf("transaction timing out despite backend being commited: %v", newErr)
		}
	}()
	backend.Commit()
	time.Sleep(time.Duration(timeout) * time.Second)

	receipt, err = client.backend.TransactionReceipt(ctx, tx.Hash())
	if receipt == nil {
		t.Error("receipt not found despite transaction being mined")
	}

	tx, err = client.CreateAndSendFakeTx(ctx, 1)
	if err != nil {
		t.Error(err)
	}

	// Tests transaction timing out when the block is mined beyond
	// the timeout period

	go func() {
		newErr := client.WaitForTransaction(ctx, tx.Hash(), timeout)
		if newErr == nil {
			t.Error("transaction not timing out despite backend being committed too late")
		}
	}()
	time.Sleep(time.Duration(timeout) * time.Second)
	backend.Commit()

	tx, err = client.CreateAndSendFakeTx(ctx, 2)
	if err != nil {
		t.Error(err)
	}
	// Tests function returning an error when parent context
	// is canceled

	newCtx, cancel := context.WithCancel(ctx)
	go func() {
		newErr := client.WaitForTransaction(newCtx, tx.Hash(), timeout)
		if newErr == nil {
			t.Error("no error despite parent context being canceled")
		}
	}()
	cancel()
	newCtx.Done()
	//time.Sleep(time.Duration(timeout) * time.Second)
}
