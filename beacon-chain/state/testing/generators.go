package testing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

// GeneratePendingDeposit is used for testing and producing a signed pending deposit
func GeneratePendingDeposit(t *testing.T, key common.SecretKey, amount uint64, withdrawalCredentials [32]byte, slot primitives.Slot) *ethpb.PendingDeposit {
	dm := &ethpb.DepositMessage{
		PublicKey:             key.PublicKey().Marshal(),
		WithdrawalCredentials: withdrawalCredentials[:],
		Amount:                amount,
	}
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(dm, domain)
	require.NoError(t, err)
	sig := key.Sign(sr[:])
	depositData := &ethpb.Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(dm.PublicKey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(dm.WithdrawalCredentials),
		Amount:                dm.Amount,
		Signature:             sig.Marshal(),
	}
	valid, err := blocks.IsValidDepositSignature(depositData)
	require.NoError(t, err)
	require.Equal(t, true, valid)
	return &ethpb.PendingDeposit{
		PublicKey:             bytesutil.SafeCopyBytes(dm.PublicKey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(dm.WithdrawalCredentials),
		Amount:                dm.Amount,
		Signature:             sig.Marshal(),
		Slot:                  slot,
	}
}

// StateWithActiveBalanceETH generates a mock beacon state given a balance in eth
func StateWithActiveBalanceETH(t *testing.T, balETH uint64) state.BeaconState {
	gwei := balETH * 1_000_000_000
	balPerVal := params.BeaconConfig().MinActivationBalance
	numVals := gwei / balPerVal

	vals := make([]*ethpb.Validator, numVals)
	bals := make([]uint64, numVals)
	for i := uint64(0); i < numVals; i++ {
		wc := make([]byte, 32)
		wc[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		wc[31] = byte(i)
		vals[i] = &ethpb.Validator{
			ActivationEpoch:       0,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      balPerVal,
			WithdrawalCredentials: wc,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		}
		bals[i] = balPerVal
	}
	st, err := state_native.InitializeFromProtoUnsafeElectra(&ethpb.BeaconStateElectra{
		Slot:       10 * params.BeaconConfig().SlotsPerEpoch,
		Validators: vals,
		Balances:   bals,
		Fork: &ethpb.Fork{
			CurrentVersion: params.BeaconConfig().ElectraForkVersion,
		},
	})
	require.NoError(t, err)
	// set some fake finalized checkpoint
	require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  make([]byte, 32),
	}))
	return st
}

// StateWithPendingDeposits with pending deposits and existing ethbalance
func StateWithPendingDeposits(t *testing.T, balETH uint64, numDeposits, amount uint64) state.BeaconState {
	st := StateWithActiveBalanceETH(t, balETH)
	deps := make([]*ethpb.PendingDeposit, numDeposits)
	validators := st.Validators()
	for i := 0; i < len(deps); i += 1 {
		sk, err := bls.RandKey()
		require.NoError(t, err)
		wc := make([]byte, 32)
		wc[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		wc[31] = byte(i)
		validators[i].PublicKey = sk.PublicKey().Marshal()
		validators[i].WithdrawalCredentials = wc
		deps[i] = GeneratePendingDeposit(t, sk, amount, bytesutil.ToBytes32(wc), 0)
	}
	require.NoError(t, st.SetValidators(validators))
	st.SaveValidatorIndices()
	require.NoError(t, st.SetPendingDeposits(deps))
	return st
}
