package depositcontract_test

import (
	"context"
	"encoding/binary"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	depositcontract "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/interop"
)

func TestSetupRegistrationContract_OK(t *testing.T) {
	_, err := depositcontract.Setup()
	if err != nil {
		log.Fatalf("Can not deploy validator registration contract: %v", err)
	}
}

// negative test case, deposit with less than 1 ETH which is less than the top off amount.
func TestRegister_Below1ETH(t *testing.T) {
	testAccount, err := depositcontract.Setup()
	if err != nil {
		t.Fatal(err)
	}

	// Generate deposit data
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, 1)
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}

	var depositDataRoot [32]byte
	copy(depositDataRoot[:], depositDataRoots[0])
	testAccount.TxOpts.Value = depositcontract.LessThan1Eth()
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubKeys[0].Marshal(), depositDataItems[0].WithdrawalCredentials, depositDataItems[0].Signature, depositDataRoot)
	if err == nil {
		t.Error("Validator registration should have failed with insufficient deposit")
	}
}

// normal test case, test depositing 32 ETH and verify HashChainValue event is correctly emitted.
func TestValidatorRegister_OK(t *testing.T) {
	testAccount, err := depositcontract.Setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.TxOpts.Value = depositcontract.Amount32Eth()

	// Generate deposit data
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, 1)
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}

	var depositDataRoot [32]byte
	copy(depositDataRoot[:], depositDataRoots[0])
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubKeys[0].Marshal(), depositDataItems[0].WithdrawalCredentials, depositDataItems[0].Signature, depositDataRoot)
	testAccount.Backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubKeys[0].Marshal(), depositDataItems[0].WithdrawalCredentials, depositDataItems[0].Signature, depositDataRoot)
	testAccount.Backend.Commit()
	if err != nil {
		t.Errorf("Validator registration failed: %v", err)
	}
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubKeys[0].Marshal(), depositDataItems[0].WithdrawalCredentials, depositDataItems[0].Signature, depositDataRoot)
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
		_, _, _, _, idx, err := depositcontract.UnpackDepositLogData(log.Data)
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
	testAccount, err := depositcontract.Setup()
	if err != nil {
		t.Fatal(err)
	}
	testAccount.TxOpts.Value = depositcontract.Amount32Eth()

	// Generate deposit data
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, 1)
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}

	var depositDataRoot [32]byte
	copy(depositDataRoot[:], depositDataRoots[0])
	_, err = testAccount.Contract.Deposit(testAccount.TxOpts, pubKeys[0].Marshal(), depositDataItems[0].WithdrawalCredentials, depositDataItems[0].Signature, depositDataRoot)
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
	if bal.Cmp(depositcontract.Amount32Eth()) != 0 {
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
