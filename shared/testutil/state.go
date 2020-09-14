package testutil

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var st, _ = stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{
	BlockRoots:                 filledByteSlice2D(params.BeaconConfig().SlotsPerHistoricalRoot, 32),
	StateRoots:                 filledByteSlice2D(params.BeaconConfig().SlotsPerHistoricalRoot, 32),
	Slashings:                  make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
	RandaoMixes:                filledByteSlice2D(params.BeaconConfig().EpochsPerHistoricalVector, 32),
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
	Eth1DataVotes:       make([]*ethpb.Eth1Data, 0),
	HistoricalRoots:     make([][]byte, 0),
	JustificationBits:   bitfield.Bitvector4{0x0},
	FinalizedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
	LatestBlockHeader: &ethpb.BeaconBlockHeader{
		ParentRoot: make([]byte, 32),
		StateRoot:  make([]byte, 32),
		BodyRoot:   make([]byte, 32),
	},
	PreviousEpochAttestations:   make([]*pb.PendingAttestation, 0),
	CurrentEpochAttestations:    make([]*pb.PendingAttestation, 0),
	PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
})

// NewBeaconState creates a beacon state with minimum marshalable fields.
func NewBeaconState() *stateTrie.BeaconState {
	return st.Copy()
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
