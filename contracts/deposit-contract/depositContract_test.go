package depositcontract

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	amount33Eth, _        = new(big.Int).SetString("33000000000000000000", 10)
	amount32Eth, _        = new(big.Int).SetString("32000000000000000000", 10)
	amountLessThan1Eth, _ = new(big.Int).SetString("500000000000000000", 10)
)

type testAccount struct {
	addr         common.Address
	contract     *DepositContract
	contractAddr common.Address
	backend      *backends.SimulatedBackend
	txOpts       *bind.TransactOpts
}

func setup() (*testAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, _ := crypto.GenerateKey()
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// strip off the 0x and the first 2 characters 04 which is always the EC prefix and is not required.
	publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
	var pubKey = make([]byte, 48)
	copy(pubKey[:], []byte(publicKeyBytes))

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	txOpts := bind.NewKeyedTransactor(privKey)
	startingBalance, _ := new(big.Int).SetString("100000000000000000000000000000000000000", 10)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	backend := backends.NewSimulatedBackend(genesis, 210000000000)

	depositsRequired := big.NewInt(8)
	minDeposit := big.NewInt(1e9)
	maxDeposit := big.NewInt(32e9)
	contractAddr, _, contract, err := DeployDepositContract(txOpts, backend, depositsRequired, minDeposit, maxDeposit, big.NewInt(1), addr)
	if err != nil {
		return nil, err
	}

	return &testAccount{addr, contract, contractAddr, backend, txOpts}, nil
}

func TestSetupRegistrationContract_OK(t *testing.T) {
	_, err := setup()
	if err != nil {
		log.Fatalf("Can not deploy validator registration contract: %v", err)
	}
}

// negative test case, deposit with less than 1 ETH which is less than the top off amount.
func TestRegister_Below1ETH(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	testAccount.txOpts.Value = amountLessThan1Eth
	_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{})
	if err == nil {
		t.Error("Validator registration should have failed with insufficient deposit")
	}
}

// negative test case, deposit with more than 32 ETH which is more than the asked amount.
func TestRegister_Above32Eth(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	testAccount.txOpts.Value = amount33Eth
	_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{})
	if err == nil {
		t.Error("Validator registration should have failed with more than asked deposit amount")
	}
}

// normal test case, test depositing 32 ETH and verify HashChainValue event is correctly emitted.
func TestValidatorRegister_OK(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.txOpts.Value = amount32Eth

	_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{'A'})
	testAccount.backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}
	_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{'B'})
	testAccount.backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}
	_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{'C'})
	testAccount.backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			testAccount.contractAddr,
		},
	}

	logs, err := testAccount.backend.FilterLogs(context.Background(), query)
	if err != nil {
		t.Fatalf("Unable to get logs of deposit contract: %v", err)
	}

	merkleTreeIndex := make([]uint64, 5)
	depositData := make([][]byte, 5)

	for i, log := range logs {
		_, data, idx, _, err := UnpackDepositLogData(log.Data)
		if err != nil {
			t.Fatalf("Unable to unpack log data: %v", err)
		}
		merkleTreeIndex[i] = binary.LittleEndian.Uint64(idx)
		depositData[i] = data
	}

	if merkleTreeIndex[0] != 0 {
		t.Errorf("Deposit event total desposit count miss matched. Want: %d, Got: %d", 1, merkleTreeIndex[0])
	}

	if merkleTreeIndex[1] != 1 {
		t.Errorf("Deposit event total desposit count miss matched. Want: %d, Got: %d", 2, merkleTreeIndex[1])
	}

	if merkleTreeIndex[2] != 2 {
		t.Errorf("Deposit event total desposit count miss matched. Want: %v, Got: %v", 3, merkleTreeIndex[2])
	}
}

// normal test case, test beacon chain start log event.
func TestChainStart_OK(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.txOpts.Value = amount32Eth

	for i := 0; i < 8; i++ {
		_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{'A'})
		if err != nil {
			t.Errorf("Validator registration failed: %v", err)
		}
	}

	testAccount.backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			testAccount.contractAddr,
		},
	}

	logs, err := testAccount.backend.FilterLogs(context.Background(), query)
	if err != nil {
		t.Fatalf("Unable to get logs %v", err)
	}

	if logs[8].Topics[0] != hashutil.Hash([]byte("ChainStart(bytes32,bytes)")) {
		t.Error("Chain start even did not get emitted")
	}
}

func TestDrain(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.txOpts.Value = amount32Eth

	_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{'A'})
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}

	testAccount.backend.Commit()

	ctx := context.Background()
	bal, err := testAccount.backend.BalanceAt(ctx, testAccount.contractAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal.Cmp(amount32Eth) != 0 {
		t.Fatal("deposit didnt work")
	}

	testAccount.txOpts.Value = big.NewInt(0)
	if _, err := testAccount.contract.Drain(testAccount.txOpts); err != nil {
		t.Fatal(err)
	}

	testAccount.backend.Commit()

	bal, err = testAccount.backend.BalanceAt(ctx, testAccount.contractAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(0).Cmp(bal) != 0 {
		t.Errorf("Drain did not drain balance: %v", bal)
	}
}
