package altair

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// DeterministicGenesisStateAltair returns a genesis state in hard fork 1 format made using the deterministic deposits.
func DeterministicGenesisStateAltair(t testing.TB, numValidators uint64) (iface.BeaconStateAltair, []bls.SecretKey) {
	deposits, privKeys, err := testutil.DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get %d deposits", numValidators))
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators))
	}
	beaconState, err := altair.GenesisBeaconState(context.Background(), deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators))
	}
	testutil.ResetCache()
	return beaconState, privKeys
}
