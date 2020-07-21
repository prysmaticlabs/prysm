package depositutil_test

import (
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDepositInput_GeneratesPb(t *testing.T) {
	k1 := bls.RandKey()
	k2 := bls.RandKey()

	result, _, err := depositutil.DepositInput(k1, k2, 0)
	require.NoError(t, err)
	assert.DeepEqual(t, k1.PublicKey().Marshal(), result.PublicKey)

	sig, err := bls.SignatureFromBytes(result.Signature)
	require.NoError(t, err)
	sr, err := ssz.SigningRoot(result)
	require.NoError(t, err)
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		nil, /*forkVersion*/
		nil, /*genesisValidatorsRoot*/
	)
	require.NoError(t, err)
	root, err := ssz.HashTreeRoot(&pb.SigningData{ObjectRoot: sr[:], Domain: domain[:]})
	require.NoError(t, err)
	assert.Equal(t, true, sig.Verify(k1.PublicKey(), root[:]))
}
