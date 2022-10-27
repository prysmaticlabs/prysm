package state_native_test

import (
	"testing"

	statenative "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestReadOnlyValidator_ReturnsErrorOnNil(t *testing.T) {
	if _, err := statenative.NewValidator(nil); err != statenative.ErrNilWrappedValidator {
		t.Errorf("Wrong error returned. Got %v, wanted %v", err, statenative.ErrNilWrappedValidator)
	}
}

func TestReadOnlyValidator_EffectiveBalance(t *testing.T) {
	bal := uint64(234)
	v, err := statenative.NewValidator(&ethpb.Validator{EffectiveBalance: bal})
	require.NoError(t, err)
	assert.Equal(t, bal, v.EffectiveBalance())
}

func TestReadOnlyValidator_ActivationEligibilityEpoch(t *testing.T) {
	epoch := types.Epoch(234)
	v, err := statenative.NewValidator(&ethpb.Validator{ActivationEligibilityEpoch: epoch})
	require.NoError(t, err)
	assert.Equal(t, epoch, v.ActivationEligibilityEpoch())
}

func TestReadOnlyValidator_ActivationEpoch(t *testing.T) {
	epoch := types.Epoch(234)
	v, err := statenative.NewValidator(&ethpb.Validator{ActivationEpoch: epoch})
	require.NoError(t, err)
	assert.Equal(t, epoch, v.ActivationEpoch())
}

func TestReadOnlyValidator_WithdrawableEpoch(t *testing.T) {
	epoch := types.Epoch(234)
	v, err := statenative.NewValidator(&ethpb.Validator{WithdrawableEpoch: epoch})
	require.NoError(t, err)
	assert.Equal(t, epoch, v.WithdrawableEpoch())
}

func TestReadOnlyValidator_ExitEpoch(t *testing.T) {
	epoch := types.Epoch(234)
	v, err := statenative.NewValidator(&ethpb.Validator{ExitEpoch: epoch})
	require.NoError(t, err)
	assert.Equal(t, epoch, v.ExitEpoch())
}

func TestReadOnlyValidator_PublicKey(t *testing.T) {
	key := [fieldparams.BLSPubkeyLength]byte{0xFA, 0xCC}
	v, err := statenative.NewValidator(&ethpb.Validator{PublicKey: key[:]})
	require.NoError(t, err)
	assert.Equal(t, key, v.PublicKey())
}

func TestReadOnlyValidator_WithdrawalCredentials(t *testing.T) {
	creds := []byte{0xFA, 0xCC}
	v, err := statenative.NewValidator(&ethpb.Validator{WithdrawalCredentials: creds})
	require.NoError(t, err)
	assert.DeepEqual(t, creds, v.WithdrawalCredentials())
}

func TestReadOnlyValidator_HasETH1WithdrawalCredentials(t *testing.T) {
	creds := []byte{0xFA, 0xCC}
	v, err := statenative.NewValidator(&ethpb.Validator{WithdrawalCredentials: creds})
	require.NoError(t, err)
	require.Equal(t, false, v.HasETH1WithdrawalCredential())
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = statenative.NewValidator(&ethpb.Validator{WithdrawalCredentials: creds})
	require.NoError(t, err)
	require.Equal(t, true, v.HasETH1WithdrawalCredential())
	// No Withdrawal cred
	v, err = statenative.NewValidator(&ethpb.Validator{})
	require.NoError(t, err)
	require.Equal(t, false, v.HasETH1WithdrawalCredential())
}

func TestReadOnlyValidator_IsFullyWithdrawable(t *testing.T) {
	// No ETH1 prefix
	creds := []byte{0xFA, 0xCC}
	v, err := statenative.NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		WithdrawableEpoch:     2,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsFullyWithdrawable(3))
	// Wrong withdrawable epoch
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = statenative.NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		WithdrawableEpoch:     2,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsFullyWithdrawable(1))
	// Fully withdrawable
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = statenative.NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		WithdrawableEpoch:     2,
	})
	require.NoError(t, err)
	require.Equal(t, true, v.IsFullyWithdrawable(3))
}

func TestReadOnlyValidator_IsPartiallyWithdrawable(t *testing.T) {
	// No ETH1 prefix
	creds := []byte{0xFA, 0xCC}
	v, err := statenative.NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance+1))
	// Not the right effective balance
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = statenative.NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance - 1,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance+1))
	// Not enough balance
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = statenative.NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance))
	// Partially Withdrawable
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = statenative.NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
	})
	require.NoError(t, err)
	require.Equal(t, true, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance+1))
}

func TestReadOnlyValidator_Slashed(t *testing.T) {
	v, err := statenative.NewValidator(&ethpb.Validator{Slashed: true})
	require.NoError(t, err)
	assert.Equal(t, true, v.Slashed())
}
