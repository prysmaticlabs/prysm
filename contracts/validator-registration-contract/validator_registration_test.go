package vrc

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	amount33Eth, _        = new(big.Int).SetString("33000000000000000000", 10)
	amount32Eth, _        = new(big.Int).SetString("32000000000000000000", 10)
	amountLessThan1Eth, _ = new(big.Int).SetString("500000000000000000", 10)
)

type testAccount struct {
	addr     common.Address
	contract *ValidatorRegistration
	backend  *backends.SimulatedBackend
	txOpts   *bind.TransactOpts
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
	startingBalance, _ := new(big.Int).SetString("1000000000000000000000", 10)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	backend := backends.NewSimulatedBackend(genesis, 2100000)

	_, _, contract, err := DeployValidatorRegistration(txOpts, backend)
	if err != nil {
		return nil, err
	}

	return &testAccount{addr, contract, backend, txOpts}, nil
}

func TestSetupAndContractRegistration(t *testing.T) {
	_, err := setup()
	if err != nil {
		log.Fatalf("Can not deploy validator registration contract: %v", err)
	}
}

// negative test case, deposit with less than 1 ETH which is less than the top off amount.
func TestRegisterWithLessThan1Eth(t *testing.T) {
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
func TestRegisterWithMoreThan32Eth(t *testing.T) {
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
func TestValidatorRegisters(t *testing.T) {
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
	log, err := testAccount.contract.FilterDeposit(&bind.FilterOpts{}, [][]byte{})

	defer func() {
		err = log.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if err != nil {
		t.Fatal(err)
	}
	if log.Error() != nil {
		t.Fatal(log.Error())
	}
	log.Next()

	index := make([]byte, 8)
	binary.BigEndian.PutUint64(index, 65536)

	if !bytes.Equal(log.Event.MerkleTreeIndex, index) {
		t.Errorf("HashChainValue event total desposit count miss matched. Want: %v, Got: %v", index, log.Event.MerkleTreeIndex)
	}
	if !bytes.Equal(log.Event.Data[len(log.Event.Data)-1:], []byte{'A'}) {
		t.Errorf("validatorRegistered event randao commitment miss matched. Want: %v, Got: %v", []byte{'A'}, log.Event.Data[len(log.Event.Data)-1:])
	}

	log.Next()
	binary.BigEndian.PutUint64(index, 65537)
	if !bytes.Equal(log.Event.MerkleTreeIndex, index) {
		t.Errorf("HashChainValue event total desposit count miss matched. Want: %v, Got: %v", index, log.Event.MerkleTreeIndex)
	}
	if !bytes.Equal(log.Event.Data[len(log.Event.Data)-1:], []byte{'B'}) {
		t.Errorf("validatorRegistered event randao commitment miss matched. Want: %v, Got: %v", []byte{'B'}, log.Event.Data[len(log.Event.Data)-1:])
	}

	log.Next()
	binary.BigEndian.PutUint64(index, 65538)
	if !bytes.Equal(log.Event.MerkleTreeIndex, index) {
		t.Errorf("HashChainValue event total desposit count miss matched. Want: %v, Got: %v", index, log.Event.MerkleTreeIndex)
	}
	if !bytes.Equal(log.Event.Data[len(log.Event.Data)-1:], []byte{'C'}) {
		t.Errorf("validatorRegistered event randao commitment miss matched. Want: %v, Got: %v", []byte{'B'}, log.Event.Data[len(log.Event.Data)-1:])
	}
}

// normal test case, test beacon chain start log event.
func TestChainStarts(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.txOpts.Value = amount32Eth

	for i := 0; i < 9; i++ {
		_, err = testAccount.contract.Deposit(testAccount.txOpts, []byte{'A'})
		testAccount.backend.Commit()
		if err != nil {
			t.Errorf("Validator registration failed: %v", err)
		}
	}

	log, err := testAccount.contract.FilterChainStart(&bind.FilterOpts{}, [][]byte{})

	defer func() {
		err = log.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if err != nil {
		t.Fatal(err)
	}
	if log.Error() != nil {
		t.Fatal(log.Error())
	}
	log.Next()

	if len(log.Event.Time) == 0 {
		t.Error("Chain start even did not get emitted, The start time is empty")
	}
}
