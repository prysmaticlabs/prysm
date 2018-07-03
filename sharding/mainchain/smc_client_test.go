package mainchain

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/internal"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

// Verifies that SMCCLient implements the sharding Service inteface.
var _ = sharding.Service(&SMCClient{})

// TestWaitForTransaction tests the WaitForTransaction function in the smcClient, however since for testing we do not have
// an inmemory rpc server to interact with the simulated backend, we have to define the function in the test rather than
// in `smc_client.go`.
// TODO: Test the function in the main file instead of defining it here if a rpc server for testing can be
// implemented.
func TestWaitForTransaction_TransactionNotMined(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}
	ctx := context.Background()
	timeout := time.Duration(1)

	tx, err := client.CreateAndSendFakeTx(ctx)
	if err != nil {
		t.Error(err)
	}

	receipt, err := client.Backend.TransactionReceipt(ctx, tx.Hash())
	if receipt != nil {
		t.Errorf("transaction mined despite backend not being committed: %v", receipt)
	}
	err = client.WaitForTransactionTimedOut(ctx, tx.Hash(), timeout)
	if err == nil {
		t.Error("transaction is supposed to timeout and return a error")
	}

}

func TestWaitForTransaction_IsMinedImmediately(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}
	ctx := context.Background()
	timeout := time.Duration(1)
	var wg sync.WaitGroup
	wg.Add(1)

	tx, err := client.CreateAndSendFakeTx(ctx)
	if err != nil {
		t.Error(err)
	}

	// Tests transaction timing out when the block is mined immediately
	// in the timeout period
	go func() {
		newErr := client.WaitForTransaction(ctx, tx.Hash(), timeout)
		if newErr != nil {
			t.Errorf("transaction timing out despite backend being committed: %v", newErr)
		}
		receipt, err := client.Backend.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			t.Errorf("receipt could not be retrieved:%v", err)
		}
		if receipt == nil {
			t.Error("receipt not found despite transaction being mined")
		}
		wg.Done()
	}()
	client.Backend.Commit()
	wg.Wait()

}
func TestWaitForTransaction_TimesOut(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}
	ctx := context.Background()
	timeout := time.Duration(1)
	var wg sync.WaitGroup
	wg.Add(1)

	tx, err := client.CreateAndSendFakeTx(ctx)
	if err != nil {
		t.Error(err)
	}

	go func() {
		newErr := client.WaitForTransactionTimedOut(ctx, tx.Hash(), timeout)
		if newErr == nil {
			t.Error("transaction not timing out despite backend being committed too late")
		}
		client.Backend.Commit()
		wg.Done()
	}()
	wg.Wait()

}
func TestWaitForTransaction_IsCancelledWhenParentCtxCancelled(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}
	ctx := context.Background()
	timeout := time.Duration(1)
	var wg sync.WaitGroup
	wg.Add(1)

	tx, err := client.CreateAndSendFakeTx(ctx)
	if err != nil {
		t.Error(err)
	}

	newCtx, cancel := context.WithCancel(ctx)
	go func() {
		newErr := client.WaitForTransaction(newCtx, tx.Hash(), timeout)
		if newErr == nil {
			t.Error("no error despite parent context being canceled")
		}
		wg.Done()
	}()
	cancel()
	newCtx.Done()

	wg.Wait()
}
