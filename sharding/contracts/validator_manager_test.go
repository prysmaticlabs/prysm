package contracts

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

func TestContractCreation(t *testing.T) {
	accountBalance, _ := new(big.Int).SetString("1000000000000000000000", 10)
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance}})
	transactOpts := bind.NewKeyedTransactor(key)
	callOpts := bind.CallOpts{}
	filterOpts := bind.FilterOpts{}

	_, _, vmc, err := DeployVMC(transactOpts, contractBackend)
	contractBackend.Commit()
	if err != nil {
		t.Fatalf("can't deploy VMC: %v", err)
	}

	// Test if collation gas limit is a 10000000
	gasLimit, err := vmc.GetCollationGasLimit(&callOpts)
	if gasLimit.Cmp(new(big.Int).SetInt64(10000000)) != 0 {
		t.Fatalf("collation gas limit should be 10000000 gas")
	}

	// Test deposit() function
	// Deposit 100 Eth
	transactOpts.Value, _ = new(big.Int).SetString("100000000000000000000", 10)

	if _, err := vmc.Deposit(transactOpts); err != nil {
		t.Fatalf("Validator cannot deposit: %v", err)
	}
	contractBackend.Commit()
	//Check for the Deposit event
	depositsIterator, err := vmc.FilterDeposit(&filterOpts)
	if err != nil {
		t.Fatalf("Failed to get Deposit event: %v", err)
	}
	if depositsIterator.Next() == false {
		t.Fatal("No Deposit event found")
	}
	if depositsIterator.Event.Validator != addr {
		t.Fatalf("Validator address mismatch: %x should be %x", depositsIterator.Event.Validator, addr)
	}
	if depositsIterator.Event.Index.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Validator index mismatch: %d should be 0", depositsIterator.Event.Index)
	}
	fmt.Printf("%x\n", depositsIterator.Event.Validator)

}
