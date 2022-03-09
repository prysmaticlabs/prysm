package util

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/testing/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// FillRootsNaturalOpt is meant to be used as an option when calling NewBeaconState.
// It fills state and block roots with hex representations of natural numbers starting with 0.
// Example: 16 becomes 0x00...0f.
func FillRootsNaturalOpt(state *ethpb.BeaconState) error {
	roots, err := prepareRoots()
	if err != nil {
		return err
	}
	state.StateRoots = roots
	state.BlockRoots = roots
	return nil
}

// FillRootsNaturalOptAltair is meant to be used as an option when calling NewBeaconStateAltair.
// It fills state and block roots with hex representations of natural numbers starting with 0.
// Example: 16 becomes 0x00...0f.
func FillRootsNaturalOptAltair(state *ethpb.BeaconStateAltair) error {
	roots, err := prepareRoots()
	if err != nil {
		return err
	}
	state.StateRoots = roots
	state.BlockRoots = roots
	return nil
}

// FillRootsNaturalOptBellatrix is meant to be used as an option when calling NewBeaconStateAltair.
// It fills state and block roots with hex representations of natural numbers starting with 0.
// Example: 16 becomes 0x00...0f.
func FillRootsNaturalOptBellatrix(state *ethpb.BeaconStateBellatrix) error {
	roots, err := prepareRoots()
	if err != nil {
		return err
	}
	state.StateRoots = roots
	state.BlockRoots = roots
	return nil
}

func WithStateSlot(slot types.Slot) NewBeaconStateOption {
	return func(st *ethpb.BeaconState) error {
		st.Slot = slot
		return nil
	}
}

func WithLatestHeaderFromBlock(t *testing.T, b block.SignedBeaconBlock) NewBeaconStateOption {
	return func(st *ethpb.BeaconState) error {
		sh, err := b.Header()
		require.NoError(t, err)
		st.LatestBlockHeader = sh.Header
		return nil
	}
}

type NewBeaconStateOption func(state *ethpb.BeaconState) error

// NewBeaconState creates a beacon state with minimum marshalable fields.
func NewBeaconState(options ...NewBeaconStateOption) (state.BeaconState, error) {
	seed := &ethpb.BeaconState{
		BlockRoots:                 filledByteSlice2D(uint64(params.MainnetConfig().SlotsPerHistoricalRoot), 32),
		StateRoots:                 filledByteSlice2D(uint64(params.MainnetConfig().SlotsPerHistoricalRoot), 32),
		Slashings:                  make([]uint64, params.MainnetConfig().EpochsPerSlashingsVector),
		RandaoMixes:                filledByteSlice2D(uint64(params.MainnetConfig().EpochsPerHistoricalVector), 32),
		Validators:                 make([]*ethpb.Validator, 0),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, fieldparams.RootLength),
			BlockHash:   make([]byte, 32),
		},
		Fork: &ethpb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  make([]byte, 4),
		},
		Eth1DataVotes:               make([]*ethpb.Eth1Data, 0),
		HistoricalRoots:             make([][]byte, 0),
		JustificationBits:           bitfield.Bitvector4{0x0},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		LatestBlockHeader:           HydrateBeaconHeader(&ethpb.BeaconBlockHeader{}),
		PreviousEpochAttestations:   make([]*ethpb.PendingAttestation, 0),
		CurrentEpochAttestations:    make([]*ethpb.PendingAttestation, 0),
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
	}

	for _, opt := range options {
		err := opt(seed)
		if err != nil {
			return nil, err
		}
	}

	var st, err = v1.InitializeFromProtoUnsafe(seed)
	if err != nil {
		return nil, err
	}

	return st.Copy().(*v1.BeaconState), nil
}

// NewBeaconStateAltair creates a beacon state with minimum marshalable fields.
func NewBeaconStateAltair(options ...func(state *ethpb.BeaconStateAltair) error) (state.BeaconStateAltair, error) {
	pubkeys := make([][]byte, 512)
	for i := range pubkeys {
		pubkeys[i] = make([]byte, 48)
	}

	seed := &ethpb.BeaconStateAltair{
		BlockRoots:                 filledByteSlice2D(uint64(params.MainnetConfig().SlotsPerHistoricalRoot), 32),
		StateRoots:                 filledByteSlice2D(uint64(params.MainnetConfig().SlotsPerHistoricalRoot), 32),
		Slashings:                  make([]uint64, params.MainnetConfig().EpochsPerSlashingsVector),
		RandaoMixes:                filledByteSlice2D(uint64(params.MainnetConfig().EpochsPerHistoricalVector), 32),
		Validators:                 make([]*ethpb.Validator, 0),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, fieldparams.RootLength),
			BlockHash:   make([]byte, 32),
		},
		Fork: &ethpb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  make([]byte, 4),
		},
		Eth1DataVotes:               make([]*ethpb.Eth1Data, 0),
		HistoricalRoots:             make([][]byte, 0),
		JustificationBits:           bitfield.Bitvector4{0x0},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		LatestBlockHeader:           HydrateBeaconHeader(&ethpb.BeaconBlockHeader{}),
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		PreviousEpochParticipation:  make([]byte, 0),
		CurrentEpochParticipation:   make([]byte, 0),
		CurrentSyncCommittee: &ethpb.SyncCommittee{
			Pubkeys:         pubkeys,
			AggregatePubkey: make([]byte, 48),
		},
		NextSyncCommittee: &ethpb.SyncCommittee{
			Pubkeys:         pubkeys,
			AggregatePubkey: make([]byte, 48),
		},
	}

	for _, opt := range options {
		err := opt(seed)
		if err != nil {
			return nil, err
		}
	}

	var st, err = v2.InitializeFromProtoUnsafe(seed)
	if err != nil {
		return nil, err
	}

	return st.Copy().(*v2.BeaconState), nil
}

// NewBeaconStateBellatrix creates a beacon state with minimum marshalable fields.
func NewBeaconStateBellatrix(options ...func(state *ethpb.BeaconStateBellatrix) error) (state.BeaconStateBellatrix, error) {
	pubkeys := make([][]byte, 512)
	for i := range pubkeys {
		pubkeys[i] = make([]byte, 48)
	}

	seed := &ethpb.BeaconStateBellatrix{
		BlockRoots:                 filledByteSlice2D(uint64(params.MainnetConfig().SlotsPerHistoricalRoot), 32),
		StateRoots:                 filledByteSlice2D(uint64(params.MainnetConfig().SlotsPerHistoricalRoot), 32),
		Slashings:                  make([]uint64, params.MainnetConfig().EpochsPerSlashingsVector),
		RandaoMixes:                filledByteSlice2D(uint64(params.MainnetConfig().EpochsPerHistoricalVector), 32),
		Validators:                 make([]*ethpb.Validator, 0),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, fieldparams.RootLength),
			BlockHash:   make([]byte, 32),
		},
		Fork: &ethpb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  make([]byte, 4),
		},
		Eth1DataVotes:               make([]*ethpb.Eth1Data, 0),
		HistoricalRoots:             make([][]byte, 0),
		JustificationBits:           bitfield.Bitvector4{0x0},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		LatestBlockHeader:           HydrateBeaconHeader(&ethpb.BeaconBlockHeader{}),
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		PreviousEpochParticipation:  make([]byte, 0),
		CurrentEpochParticipation:   make([]byte, 0),
		CurrentSyncCommittee: &ethpb.SyncCommittee{
			Pubkeys:         pubkeys,
			AggregatePubkey: make([]byte, 48),
		},
		NextSyncCommittee: &ethpb.SyncCommittee{
			Pubkeys:         pubkeys,
			AggregatePubkey: make([]byte, 48),
		},
		LatestExecutionPayloadHeader: &ethpb.ExecutionPayloadHeader{
			ParentHash:       make([]byte, 32),
			FeeRecipient:     make([]byte, 20),
			StateRoot:        make([]byte, 32),
			ReceiptRoot:      make([]byte, 32),
			LogsBloom:        make([]byte, 256),
			PrevRandao:       make([]byte, 32),
			ExtraData:        make([]byte, 0),
			BaseFeePerGas:    make([]byte, 32),
			BlockHash:        make([]byte, 32),
			TransactionsRoot: make([]byte, 32),
		},
	}

	for _, opt := range options {
		err := opt(seed)
		if err != nil {
			return nil, err
		}
	}

	var st, err = v3.InitializeFromProtoUnsafe(seed)
	if err != nil {
		return nil, err
	}

	return st.Copy().(*v3.BeaconState), nil
}

// SSZ will fill 2D byte slices with their respective values, so we must fill these in too for round
// trip testing.
func filledByteSlice2D(length, innerLen uint64) [][]byte {
	b := make([][]byte, length)
	for i := uint64(0); i < length; i++ {
		b[i] = make([]byte, innerLen)
	}
	return b
}

func prepareRoots() ([][]byte, error) {
	rootsLen := params.MainnetConfig().SlotsPerHistoricalRoot
	roots := make([][]byte, rootsLen)
	for i := types.Slot(0); i < rootsLen; i++ {
		roots[i] = make([]byte, fieldparams.RootLength)
	}
	for j := 0; j < len(roots); j++ {
		// Remove '0x' prefix and left-pad '0' to have 64 chars in total.
		s := fmt.Sprintf("%064s", hexutil.EncodeUint64(uint64(j))[2:])
		h, err := hexutil.Decode("0x" + s)
		if err != nil {
			return nil, err
		}
		roots[j] = h
	}
	return roots, nil
}
