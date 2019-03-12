package main

import (
	"testing"
	"time"

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
	beacons := e2etestutil.NewBeaconNodes(t, 1, geth)
	beacons.Start()
	defer beacons.Stop()
	time.Sleep(5 * time.Second) // wait for beacon nodes to boot up
	// Start validators
	validators := e2etestutil.NewValidators(t, 8, beacons)
	validators.Start()
	defer validators.Stop()
	// Send deposits
	// Wait for advancement to slot X or no block produced in 30 seconds
	// Dump state
	// Report balances to log
	// Report PASS/FAIL

	_ = contractAddr

	time.Sleep(1 * time.Minute)
	t.Log("Done.")
}
