package depositcontract

import (
	"context"
	"encoding/binary"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func TestSetupRegistrationContract_OK(t *testing.T) {
	_, err := Setup()
	if err != nil {
		log.Fatalf("Can not deploy validator registration contract: %v", err)
	}
}

// negative test case, deposit with less than 1 ETH which is less than the top off amount.
func TestRegister_Below1ETH(t *testing.T) {
	testAccount, err := Setup()
	if err != nil {
		t.Fatal(err)
	}

	testAccount.TxOpts.Value = LessThan1Eth()
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, []byte{}, []byte{}, []byte{})
	if err == nil {
		t.Error("Validator registration should have failed with insufficient deposit")
	}
}

// negative test case, deposit with more than 32 ETH which is more than the asked amount.
func TestRegister_Above32Eth(t *testing.T) {
	testAccount, err := Setup()
	if err != nil {
		t.Fatal(err)
	}

	testAccount.TxOpts.Value = Amount33Eth()
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, []byte{}, []byte{}, []byte{})
	if err == nil {
		t.Error("Validator registration should have failed with more than asked deposit amount")
	}
}

// normal test case, test depositing 32 ETH and verify HashChainValue event is correctly emitted.
func TestValidatorRegister_OK(t *testing.T) {
	testAccount, err := Setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.TxOpts.Value = Amount32Eth()
	testAccount.TxOpts.GasLimit = 1000000

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte

	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubkey[:], withdrawalCreds[:], sig[:])
	testAccount.Backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubkey[:], withdrawalCreds[:], sig[:])
	testAccount.Backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubkey[:], withdrawalCreds[:], sig[:])
	testAccount.Backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			testAccount.ContractAddr,
		},
	}

	logs, err := testAccount.Backend.FilterLogs(context.Background(), query)
	if err != nil {
		t.Fatalf("Unable to get logs of deposit contract: %v", err)
	}

	merkleTreeIndex := make([]uint64, 5)

	for i, log := range logs {
		_, _, _, _, idx, err := UnpackDepositLogData(log.Data)
		if err != nil {
			t.Fatalf("Unable to unpack log data: %v", err)
		}
		merkleTreeIndex[i] = binary.LittleEndian.Uint64(idx[:])
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

func TestDrain(t *testing.T) {
	testAccount, err := Setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.TxOpts.Value = Amount32Eth()

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte

	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubkey[:], withdrawalCreds[:], sig[:])
	testAccount.Backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}

	testAccount.Backend.Commit()

	ctx := context.Background()
	bal, err := testAccount.Backend.BalanceAt(ctx, testAccount.ContractAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal.Cmp(Amount32Eth()) != 0 {
		t.Fatal("deposit didnt work")
	}

	testAccount.TxOpts.Value = big.NewInt(0)
	if _, err := testAccount.Contract.Drain(testAccount.TxOpts); err != nil {
		t.Fatal(err)
	}

	testAccount.Backend.Commit()

	bal, err = testAccount.Backend.BalanceAt(ctx, testAccount.ContractAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(0).Cmp(bal) != 0 {
		t.Errorf("Drain did not drain balance: %v", bal)
	}
}
