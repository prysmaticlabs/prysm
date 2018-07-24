package mainchain

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prysmaticlabs/prysm/shared"
)

var (
	key, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr                     = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance1001Eth, _ = new(big.Int).SetString("1001000000000000000000", 10)
)

// Verifies that SMCCLient implements the sharding Service inteface.
var _ = shared.Service(&SMCClient{})

// mockClient is struct to implement the smcClient methods for testing.
type mockClient struct {
	backend     *backends.SimulatedBackend
	blockNumber *big.Int
}

// Mirrors the function in the main file, but instead of having a client to perform rpc calls
// it is replaced by the simulated backend.
func (m *mockClient) WaitForTransaction(ctx context.Context, hash common.Hash, durationInSeconds time.Duration) error {

	var receipt *gethTypes.Receipt

	ctxTimeout, cancel := context.WithTimeout(ctx, durationInSeconds*time.Second)

	for err := error(nil); receipt == nil; receipt, err = m.backend.TransactionReceipt(ctxTimeout, hash) {

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
	return nil
}

// Creates and send Fake Transactions to the backend to be mined, takes in the context and
// the current blocknumber as an argument and returns the signed transaction after it has been sent.
func (m *mockClient) CreateAndSendFakeTx(ctx context.Context) (*gethTypes.Transaction, error) {
	tx := gethTypes.NewTransaction(m.blockNumber.Uint64(), common.HexToAddress("0x"), nil, 50000, nil, nil)
	signedtx, err := gethTypes.SignTx(tx, gethTypes.MakeSigner(&params.ChainConfig{}, m.blockNumber), key)
	if err != nil {
		return nil, err
	}
	err = m.backend.SendTransaction(ctx, signedtx)
	if err != nil {
		return nil, fmt.Errorf("unable to send transaction: %v", err)
	}
	return signedtx, nil
}

func (m *mockClient) Commit() {
	m.backend.Commit()
	m.blockNumber = big.NewInt(m.blockNumber.Int64() + 1)
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
func TestWaitForTransaction_TransactionNotMined(t *testing.T) {
	backend := setup()
	client := &mockClient{backend: backend, blockNumber: big.NewInt(0)}
	ctx := context.Background()
	timeout := time.Duration(1)

	tx, err := client.CreateAndSendFakeTx(ctx)
	if err != nil {
		t.Error(err)
	}

	receipt, err := client.backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		t.Error(err)
	}
	if receipt != nil {
		t.Errorf("transaction mined despite backend not being committed: %v", receipt)
	}
	if err = client.WaitForTransaction(ctx, tx.Hash(), timeout); err == nil {
		t.Error("transaction is supposed to timeout and return a error")
	}
}

func TestWaitForTransaction_IsMinedImmediately(t *testing.T) {
	backend := setup()
	client := &mockClient{backend: backend, blockNumber: big.NewInt(0)}
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
		receipt, err := client.backend.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			t.Errorf("receipt could not be retrieved:%v", err)
		}
		if receipt == nil {
			t.Error("receipt not found despite transaction being mined")
		}
		wg.Done()
	}()
	client.Commit()
	wg.Wait()
}
func TestWaitForTransaction_TimesOut(t *testing.T) {
	backend := setup()
	client := &mockClient{backend: backend, blockNumber: big.NewInt(0)}
	ctx := context.Background()
	timeout := time.Duration(1)
	var wg sync.WaitGroup
	wg.Add(1)

	tx, err := client.CreateAndSendFakeTx(ctx)
	if err != nil {
		t.Error(err)
	}

	go func() {
		newErr := client.WaitForTransaction(ctx, tx.Hash(), timeout)
		if newErr == nil {
			t.Error("transaction not timing out despite backend being committed too late")
		}
		client.Commit()
		wg.Done()
	}()
	wg.Wait()
}
func TestWaitForTransaction_IsCancelledWhenParentCtxCancelled(t *testing.T) {
	backend := setup()
	client := &mockClient{backend: backend, blockNumber: big.NewInt(0)}
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
