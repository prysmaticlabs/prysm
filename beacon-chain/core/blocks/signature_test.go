package blocks_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestVerifyBlockHeaderSignature(t *testing.T) {
	beaconState, err := testutil.NewBeaconState()
	require.NoError(t, err)

	privKey, err := bls.RandKey()
	require.NoError(t, err)
	validators := make([]*ethpb.Validator, 1)
	validators[0] = &ethpb.Validator{
		PublicKey:             privKey.PublicKey().Marshal(),
		WithdrawalCredentials: make([]byte, 32),
		EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
	}
	err = beaconState.SetValidators(validators)
	require.NoError(t, err)

	// Sign the block header.
	blockHeader := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          0,
			ProposerIndex: 0,
		},
	})
	domain, err := helpers.Domain(
		beaconState.Fork(),
		0,
		params.BeaconConfig().DomainBeaconProposer,
		beaconState.GenesisValidatorRoot(),
	)
	require.NoError(t, err)
	htr, err := blockHeader.Header.HashTreeRoot()
	require.NoError(t, err)
	container := &ethpb.SigningData{
		ObjectRoot: htr[:],
		Domain:     domain,
	}
	require.NoError(t, err)
	signingRoot, err := container.HashTreeRoot()
	require.NoError(t, err)

	// Set the signature in the block header.
	blockHeader.Signature = privKey.Sign(signingRoot[:]).Marshal()

	// Sig should verify.
	err = blocks.VerifyBlockHeaderSignature(beaconState, blockHeader)
	require.NoError(t, err)
}

func TestVerifyBlockSignatureUsingCurrentFork(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	bCfg := params.BeaconConfig()
	bCfg.AltairForkEpoch = 100
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.AltairForkVersion)] = 100
	params.OverrideBeaconConfig(bCfg)
	bState, keys := testutil.DeterministicGenesisState(t, 100)
	altairBlk := testutil.NewBeaconBlockAltair()
	altairBlk.Block.ProposerIndex = 0
	altairBlk.Block.Slot = params.BeaconConfig().SlotsPerEpoch * 100
	fData := &ethpb.Fork{
		Epoch:           100,
		CurrentVersion:  params.BeaconConfig().AltairForkVersion,
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
	}
	domain, err := helpers.Domain(fData, 100, params.BeaconConfig().DomainBeaconProposer, bState.GenesisValidatorRoot())
	assert.NoError(t, err)
	rt, err := helpers.ComputeSigningRoot(altairBlk.Block, domain)
	assert.NoError(t, err)
	sig := keys[0].Sign(rt[:]).Marshal()
	altairBlk.Signature = sig
	wsb, err := wrapperv2.WrappedAltairSignedBeaconBlock(altairBlk)
	require.NoError(t, err)
	assert.NoError(t, blocks.VerifyBlockSignatureUsingCurrentFork(bState, wsb))
}
