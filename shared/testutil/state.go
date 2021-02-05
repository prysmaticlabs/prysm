package testutil

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

var st, _ = stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{
	BlockRoots:                 filledRootSlice2D(params.BeaconConfig().SlotsPerHistoricalRoot),
	StateRoots:                 filledRootSlice2D(params.BeaconConfig().SlotsPerHistoricalRoot),
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
	Eth1DataVotes:               make([]*ethpb.Eth1Data, 0),
	HistoricalRoots:             make([][]byte, 0),
	JustificationBits:           bitfield.Bitvector4{0x0},
	FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, 32)},
	LatestBlockHeader:           HydrateBeaconHeader(&ethpb.BeaconBlockHeader{}),
	PreviousEpochAttestations:   make([]*pb.PendingAttestation, 0),
	CurrentEpochAttestations:    make([]*pb.PendingAttestation, 0),
	PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
})

// NewBeaconState creates a beacon state with minimum marshalable fields.
func NewBeaconState() *stateTrie.BeaconState {
	return st.Copy()
}

func filledRootSlice2D(length uint64) [][]byte {
	slice2D := filledByteSlice2D(length, 32)
	for i := uint64(0); i < uint64(len(slice2D)); i++ {
		s := fmt.Sprintf("%064s", hexutil.EncodeUint64(i)[2:])
		b, err := hexutil.Decode("0x" + s)
		if err != nil {
			log.Debug("Failed to decode root " + s)
		}
		slice2D[i] = b
	}
	return slice2D
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
