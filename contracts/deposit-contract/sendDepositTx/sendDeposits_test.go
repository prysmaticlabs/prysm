package main

import (
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
)



/*
Plan
1)create a simulated backend
2)deploy a contract  /deployContract.go
3)send deposits
4)validate that deposit log has been emitted with correct data
5)deposit log should be valid
*/

func TestEndtoEndDeposits((t *testing.T)  {
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	/* type TestAccount struct {
		Addr         common.Address
		Contract     *DepositContract
		ContractAddr common.Address
		Backend      *backends.SimulatedBackend
		TxOpts       *bind.TransactOpts
	}  */

   

}

