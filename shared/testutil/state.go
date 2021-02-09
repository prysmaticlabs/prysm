package testutil

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NewBeaconState creates a beacon state with minimum marshalable fields.
func NewBeaconState(options ...func(state *pb.BeaconState)) (*stateTrie.BeaconState, error) {
	seed := &pb.BeaconState{
		BlockRoots:                 filledByteSlice2D(params.MainnetConfig().SlotsPerHistoricalRoot, 32),
		StateRoots:                 filledByteSlice2D(params.MainnetConfig().SlotsPerHistoricalRoot, 32),
		Slashings:                  make([]uint64, params.MainnetConfig().EpochsPerSlashingsVector),
		RandaoMixes:                filledByteSlice2D(uint64(params.MainnetConfig().EpochsPerHistoricalVector), 32),
		Validators:                 make([]*ethpb.Validator, 0),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, 32),
			BlockHash:   make([]byte, 32),
		},
		Fork: &pb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  make([]byte, 4),
		},
		Eth1DataVotes:               make([]*ethpb.Eth1Data, 0),
		HistoricalRoots:             make([][]byte, 0),
		JustificationBits:           bitfield.Bitvector4{0x0},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, 32)},
		LatestBlockHeader:           HydrateBeaconHeader(&ethpb.BeaconBlockHeader{}),
		PreviousEpochAttestations:   make([]*pb.PendingAttestation, 0),
		CurrentEpochAttestations:    make([]*pb.PendingAttestation, 0),
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
	}

	for _, opt := range options {
		opt(seed)
	}

	var st, err = stateTrie.InitializeFromProtoUnsafe(seed)
	if err != nil {
		return nil, err
	}

	return st.Copy(), nil
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
