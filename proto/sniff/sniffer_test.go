package sniff

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestSlotFromBlock(t *testing.T) {
	b := testBlock()
	var slot types.Slot = 3
	b.Block.Slot = slot
	bb, err := b.MarshalSSZ()
	require.NoError(t, err)
	sfb, err := SlotFromBlock(bb)
	require.NoError(t, err)
	require.Equal(t, slot, sfb)

	ba := testBlockAltair()
	ba.Block.Slot = slot
	bab, err := ba.MarshalSSZ()
	require.NoError(t, err)
	sfba, err := SlotFromBlock(bab)
	require.NoError(t, err)
	require.Equal(t, slot, sfba)

	bm := testBlockMerge()
	bm.Block.Slot = slot
	bmb, err := ba.MarshalSSZ()
	require.NoError(t, err)
	sfbm, err := SlotFromBlock(bmb)
	require.NoError(t, err)
	require.Equal(t, slot, sfbm)
}

func testBlock() *ethpb.SignedBeaconBlock {
	return &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: types.ValidatorIndex(0),
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal:      make([]byte, 96),
				Graffiti:          make([]byte, 32),
				ProposerSlashings: []*ethpb.ProposerSlashing{},
				AttesterSlashings: []*ethpb.AttesterSlashing{},
				Attestations:      []*ethpb.Attestation{},
				Deposits:          []*ethpb.Deposit{},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{},
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
			},
		},
		Signature: make([]byte, 96),
	}
}

func testBlockAltair() *ethpb.SignedBeaconBlockAltair {
	return &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{
			ProposerIndex: types.ValidatorIndex(0),
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			Body: &ethpb.BeaconBlockBodyAltair{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
				Graffiti:          make([]byte, 32),
				ProposerSlashings: []*ethpb.ProposerSlashing{},
				AttesterSlashings: []*ethpb.AttesterSlashing{},
				Attestations:      []*ethpb.Attestation{},
				Deposits:          []*ethpb.Deposit{},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{},
				SyncAggregate: &ethpb.SyncAggregate{
					SyncCommitteeBits:      make([]byte, 64),
					SyncCommitteeSignature: make([]byte, 96),
				},
			},
		},
		Signature: make([]byte, 96),
	}
}

func testBlockMerge() *ethpb.SignedBeaconBlockMerge {
	return &ethpb.SignedBeaconBlockMerge{
		Block: &ethpb.BeaconBlockMerge{
			ProposerIndex: types.ValidatorIndex(0),
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			Body: &ethpb.BeaconBlockBodyMerge{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
				Graffiti:          make([]byte, 32),
				ProposerSlashings: []*ethpb.ProposerSlashing{},
				AttesterSlashings: []*ethpb.AttesterSlashing{},
				Attestations:      []*ethpb.Attestation{},
				Deposits:          []*ethpb.Deposit{},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{},
				SyncAggregate: &ethpb.SyncAggregate{
					SyncCommitteeBits:      make([]byte, 64),
					SyncCommitteeSignature: make([]byte, 96),
				},
				ExecutionPayload: &ethpb.ExecutionPayload{
					ParentHash:    make([]byte, 32),
					FeeRecipient:  make([]byte, 20),
					StateRoot:     make([]byte, 32),
					ReceiptRoot:   make([]byte, 32),
					LogsBloom:     make([]byte, 256),
					Random:        make([]byte, 32),
					BlockNumber:   0,
					GasLimit:      0,
					GasUsed:       0,
					Timestamp:     0,
					ExtraData:     make([]byte, 32),
					BaseFeePerGas: make([]byte, 32),
					BlockHash:     make([]byte, 32),
					Transactions:  make([][]byte, 0),
				},
			},
		},
		Signature: make([]byte, 96),
	}
}
