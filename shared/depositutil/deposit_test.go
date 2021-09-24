package depositutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDepositInput_GeneratesPb(t *testing.T) {
	k1, err := bls.RandKey()
	require.NoError(t, err)
	k2, err := bls.RandKey()
	require.NoError(t, err)

	result, _, err := depositutil.DepositInput(k1, k2, 0)
	require.NoError(t, err)
	assert.DeepEqual(t, k1.PublicKey().Marshal(), result.PublicKey)

	sig, err := bls.SignatureFromBytes(result.Signature)
	require.NoError(t, err)
	testData := &ethpb.DepositMessage{
		PublicKey:             result.PublicKey,
		WithdrawalCredentials: result.WithdrawalCredentials,
		Amount:                result.Amount,
	}
	sr, err := testData.HashTreeRoot()
	require.NoError(t, err)
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		nil, /*forkVersion*/
		nil, /*genesisValidatorsRoot*/
	)
	require.NoError(t, err)
	root, err := (&ethpb.SigningData{ObjectRoot: sr[:], Domain: domain}).HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, true, sig.Verify(k1.PublicKey(), root[:]))
}

func TestVerifyDepositSignature_ValidSig(t *testing.T) {
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	deposit := deposits[0]
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		params.BeaconConfig().GenesisForkVersion,
		params.BeaconConfig().ZeroHash[:],
	)
	require.NoError(t, err)
	err = depositutil.VerifyDepositSignature(deposit.Data, domain)
	require.NoError(t, err)
}

func TestVerifyDepositSignature_InvalidSig(t *testing.T) {
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	deposit := deposits[0]
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		params.BeaconConfig().GenesisForkVersion,
		params.BeaconConfig().ZeroHash[:],
	)
	require.NoError(t, err)
	deposit.Data.Signature = deposit.Data.Signature[1:]
	err = depositutil.VerifyDepositSignature(deposit.Data, domain)
	if err == nil {
		t.Fatal("Deposit Verification succeeds with a invalid signature")
	}
}
