package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/test/e2etestutil"
)

func TestMultiNodeDeployment(t *testing.T) {
	// Start geth dev node
	geth := e2etestutil.NewGoEthereumInstance(t)
	geth.Start()
	defer geth.Stop()
	// Deploy contract
	contractAddr := geth.DeployDepositContract()
	// Generate private keys for validator
	// Start beacon node(s)
	beacons := e2etestutil.NewBeaconNodes(t, 3, geth)
	beacons.Start()
	defer beacons.Stop()
	// Start validators
	// Send deposits
	// Wait for advancement to slot X or no block produced in 30 seconds
	// Dump state
	// Report balances to log
	// Report PASS/FAIL

	_ = contractAddr
	t.Skip()
}
