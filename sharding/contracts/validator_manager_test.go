package contracts

import (
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
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	transactOpts := bind.NewKeyedTransactor(key)
	callOpts := bind.CallOpts{}

	_, _, vmc, err := DeployVMC(transactOpts, contractBackend)
	contractBackend.Commit()
	if err != nil {
		t.Fatalf("can't deploy VMC: %v", err)
	}

	gasLimit, err := vmc.GetCollationGasLimit(&callOpts)
	if gasLimit.Cmp(new(big.Int).SetInt64(10000000)) != 0 {
		t.Fatalf("collation gas limit should be 10000000 gas")
	}
}
