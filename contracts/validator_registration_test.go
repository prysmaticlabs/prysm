package contracts

import (
	"crypto/ecdsa"
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
	amount33Eth, _ = new(big.Int).SetString("33000000000000000000", 10)
	amount32Eth, _ = new(big.Int).SetString("32000000000000000000", 10)
	amount31Eth, _ = new(big.Int).SetString("31000000000000000000", 10)
)

type testAccount struct {
	addr              common.Address
	withdrawalAddress common.Address
	randaoCommitment  [32]byte
	pubKey            [32]byte
	contract          *ValidatorRegistration
	backend           *backends.SimulatedBackend
	txOpts            *bind.TransactOpts
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
	var pubKey [32]byte
	copy(pubKey[:], []byte(publicKeyBytes))

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	txOpts := bind.NewKeyedTransactor(privKey)
	startingBalance, _ := new(big.Int).SetString("100000000000000000000", 10)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	backend := backends.NewSimulatedBackend(genesis)

	_, _, contract, err := DeployValidatorRegistration(txOpts, backend)
	if err != nil {
		return nil, err
	}

	return &testAccount{addr, common.Address{}, [32]byte{}, pubKey, contract, backend, txOpts}, nil
}

func TestSetupAndContractRegistration(t *testing.T) {
	_, err := setup()
	if err != nil {
		log.Fatalf("Can not deploy validator registration contract: %v", err)
	}
}

// negative test case, deposit with less than 32 ETH.
func TestRegisterWithLessThan32Eth(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	withdrawAddr := &common.Address{'A', 'D', 'D', 'R', 'E', 'S', 'S'}
	randaoCommitment := &[32]byte{'S', 'H', 'H', 'H', 'H', 'I', 'T', 'S', 'A', 'S', 'E', 'C', 'R', 'E', 'T'}

	testAccount.txOpts.Value = amount31Eth
	_, err = testAccount.contract.Deposit(testAccount.txOpts, testAccount.pubKey, big.NewInt(0), *withdrawAddr, *randaoCommitment)
	if err == nil {
		t.Error("Validator registration should have failed with insufficient deposit")
	}
}

// negative test case, deposit more than 32 ETH.
func TestRegisterWithMoreThan32Eth(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	withdrawAddr := &common.Address{'A', 'D', 'D', 'R', 'E', 'S', 'S'}
	randaoCommitment := &[32]byte{'S', 'H', 'H', 'H', 'H', 'I', 'T', 'S', 'A', 'S', 'E', 'C', 'R', 'E', 'T'}

	testAccount.txOpts.Value = amount33Eth
	_, err = testAccount.contract.Deposit(testAccount.txOpts, testAccount.pubKey, big.NewInt(0), *withdrawAddr, *randaoCommitment)
	if err == nil {
		t.Error("Validator registration should have failed with more than deposit amount")
	}
}

// negative test case, test registering with the same public key twice.
func TestRegisterTwice(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	withdrawAddr := &common.Address{'A', 'D', 'D', 'R', 'E', 'S', 'S'}
	randaoCommitment := &[32]byte{'S', 'H', 'H', 'H', 'H', 'I', 'T', 'S', 'A', 'S', 'E', 'C', 'R', 'E', 'T'}

	testAccount.txOpts.Value = amount32Eth
	_, err = testAccount.contract.Deposit(testAccount.txOpts, testAccount.pubKey, big.NewInt(0), *withdrawAddr, *randaoCommitment)
	testAccount.backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}

	testAccount.txOpts.Value = amount32Eth
	_, err = testAccount.contract.Deposit(testAccount.txOpts, testAccount.pubKey, big.NewInt(0), *withdrawAddr, *randaoCommitment)
	testAccount.backend.Commit()
	if err == nil {
		t.Errorf("Registration should have failed with same public key twice")
	}
}

// normal test case, test depositing 32 ETH and verify validatorRegistered event is correctly emitted.
func TestRegister(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	withdrawAddr := &common.Address{'A', 'D', 'D', 'R', 'E', 'S', 'S'}
	randaoCommitment := &[32]byte{'S', 'H', 'H', 'H', 'H', 'I', 'T', 'S', 'A', 'S', 'E', 'C', 'R', 'E', 'T'}
	shardID := big.NewInt(99)
	testAccount.txOpts.Value = amount32Eth

	_, err = testAccount.contract.Deposit(testAccount.txOpts, testAccount.pubKey, shardID, *withdrawAddr, *randaoCommitment)
	testAccount.backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}
	log, err := testAccount.contract.FilterValidatorRegistered(&bind.FilterOpts{})
	if err != nil {
		t.Fatal(err)
	}
	log.Next()
	if log.Event.WithdrawalShardID.Cmp(shardID) != 0 {
		t.Errorf("validatorRegistered event withdrawal shard ID miss matched. Want: %v, Got: %v", shardID, log.Event.WithdrawalShardID)
	}
	if log.Event.RandaoCommitment != *randaoCommitment {
		t.Errorf("validatorRegistered event randao commitment miss matched. Want: %v, Got: %v", *randaoCommitment, log.Event.RandaoCommitment)
	}
	if log.Event.PubKey != testAccount.pubKey {
		t.Errorf("validatorRegistered event public key miss matched. Want: %v, Got: %v", testAccount.pubKey, log.Event.PubKey)
	}
	if log.Event.WithdrawalAddressbytes32 != *withdrawAddr {
		t.Errorf("validatorRegistered event withdrawal address miss matched. Want: %v, Got: %v", *withdrawAddr, log.Event.WithdrawalAddressbytes32)
	}
}
