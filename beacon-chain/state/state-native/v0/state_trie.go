package v0

import (
	"context"
	"encoding/binary"
	"runtime"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v0/fieldtrie"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/runtime/version"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
	v0types "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v0/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

// InitializeFromProtoPhase0 the beacon state from a protobuf representation.
func InitializeFromProtoPhase0(st *ethpb.BeaconState) (state.BeaconState, error) {
	return InitializeFromProtoUnsafePhase0(proto.Clone(st).(*ethpb.BeaconState))
}

// InitializeFromProtoAltair the beacon state from a protobuf representation.
func InitializeFromProtoAltair(st *ethpb.BeaconStateAltair) (state.BeaconState, error) {
	return InitializeFromProtoUnsafeAltair(proto.Clone(st).(*ethpb.BeaconStateAltair))
}

// InitializeFromProtoBellatrix the beacon state from a protobuf representation.
func InitializeFromProtoBellatrix(st *ethpb.BeaconStateBellatrix) (state.BeaconState, error) {
	return InitializeFromProtoUnsafeBellatrix(proto.Clone(st).(*ethpb.BeaconStateBellatrix))
}

// InitializeFromProtoUnsafePhase0 directly uses the beacon state protobuf fields
// and sets them as fields of the BeaconState type.
func InitializeFromProtoUnsafePhase0(st *ethpb.BeaconState) (state.BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	var bRoots customtypes.BlockRoots
	for i, r := range st.BlockRoots {
		copy(bRoots[i][:], r)
	}
	var sRoots customtypes.StateRoots
	for i, r := range st.StateRoots {
		copy(sRoots[i][:], r)
	}
	hRoots := customtypes.HistoricalRoots(make([][32]byte, len(st.HistoricalRoots)))
	for i, r := range st.HistoricalRoots {
		copy(hRoots[i][:], r)
	}
	var mixes customtypes.RandaoMixes
	for i, m := range st.RandaoMixes {
		copy(mixes[i][:], m)
	}

	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	b := &BeaconState{
		version:                     Phase0,
		genesisTime:                 st.GenesisTime,
		genesisValidatorsRoot:       bytesutil.ToBytes32(st.GenesisValidatorsRoot),
		slot:                        st.Slot,
		fork:                        st.Fork,
		latestBlockHeader:           st.LatestBlockHeader,
		blockRoots:                  &bRoots,
		stateRoots:                  &sRoots,
		historicalRoots:             hRoots,
		eth1Data:                    st.Eth1Data,
		eth1DataVotes:               st.Eth1DataVotes,
		eth1DepositIndex:            st.Eth1DepositIndex,
		validators:                  st.Validators,
		balances:                    st.Balances,
		randaoMixes:                 &mixes,
		slashings:                   st.Slashings,
		previousEpochAttestations:   st.PreviousEpochAttestations,
		currentEpochAttestations:    st.CurrentEpochAttestations,
		justificationBits:           st.JustificationBits,
		previousJustifiedCheckpoint: st.PreviousJustifiedCheckpoint,
		currentJustifiedCheckpoint:  st.CurrentJustifiedCheckpoint,
		finalizedCheckpoint:         st.FinalizedCheckpoint,

		dirtyFields:           make(map[int]bool, fieldCount),
		dirtyIndices:          make(map[int][]uint64, fieldCount),
		stateFieldLeaves:      make(map[int]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[int]*stateutil.Reference, 10),
		rebuildTrie:           make(map[int]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	b.fieldIndexes = make(map[int]v0types.FieldIndex, fieldCount)
	b.fieldIndexes[0] = v0types.GenesisTime
	b.fieldIndexes[1] = v0types.GenesisValidatorsRoot
	b.fieldIndexes[2] = v0types.Slot
	b.fieldIndexes[3] = v0types.Fork
	b.fieldIndexes[4] = v0types.LatestBlockHeader
	b.fieldIndexes[5] = v0types.BlockRoots
	b.fieldIndexes[6] = v0types.StateRoots
	b.fieldIndexes[7] = v0types.HistoricalRoots
	b.fieldIndexes[8] = v0types.Eth1Data
	b.fieldIndexes[9] = v0types.Eth1DataVotes
	b.fieldIndexes[10] = v0types.Eth1DepositIndex
	b.fieldIndexes[11] = v0types.Validators
	b.fieldIndexes[12] = v0types.Balances
	b.fieldIndexes[13] = v0types.RandaoMixes
	b.fieldIndexes[14] = v0types.Slashings
	b.fieldIndexes[15] = v0types.PreviousEpochAttestations
	b.fieldIndexes[16] = v0types.CurrentEpochAttestations
	b.fieldIndexes[17] = v0types.JustificationBits
	b.fieldIndexes[18] = v0types.PreviousJustifiedCheckpoint
	b.fieldIndexes[19] = v0types.CurrentJustifiedCheckpoint
	b.fieldIndexes[20] = v0types.FinalizedCheckpoint
	b.fieldIndexesRev = make(map[v0types.FieldIndex]int, fieldCount)
	b.fieldIndexesRev[v0types.GenesisTime] = 0
	b.fieldIndexesRev[v0types.GenesisValidatorsRoot] = 1
	b.fieldIndexesRev[v0types.Slot] = 2
	b.fieldIndexesRev[v0types.Fork] = 3
	b.fieldIndexesRev[v0types.LatestBlockHeader] = 4
	b.fieldIndexesRev[v0types.BlockRoots] = 5
	b.fieldIndexesRev[v0types.StateRoots] = 6
	b.fieldIndexesRev[v0types.HistoricalRoots] = 7
	b.fieldIndexesRev[v0types.Eth1Data] = 8
	b.fieldIndexesRev[v0types.Eth1DataVotes] = 9
	b.fieldIndexesRev[v0types.Eth1DepositIndex] = 10
	b.fieldIndexesRev[v0types.Validators] = 11
	b.fieldIndexesRev[v0types.Balances] = 12
	b.fieldIndexesRev[v0types.RandaoMixes] = 13
	b.fieldIndexesRev[v0types.Slashings] = 14
	b.fieldIndexesRev[v0types.PreviousEpochAttestations] = 15
	b.fieldIndexesRev[v0types.CurrentEpochAttestations] = 16
	b.fieldIndexesRev[v0types.JustificationBits] = 17
	b.fieldIndexesRev[v0types.PreviousJustifiedCheckpoint] = 18
	b.fieldIndexesRev[v0types.CurrentJustifiedCheckpoint] = 19
	b.fieldIndexesRev[v0types.FinalizedCheckpoint] = 20

	var err error
	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[i] = true
		b.rebuildTrie[i] = true
		b.dirtyIndices[i] = []uint64{}
		b.stateFieldLeaves[i], err = fieldtrie.NewFieldTrie(v0types.FieldIndex(i), types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.BlockRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.StateRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.HistoricalRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Eth1DataVotes]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Validators]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Balances]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.RandaoMixes]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Slashings]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochAttestations]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochAttestations]] = stateutil.NewRef(1)

	state.StateCount.Inc()
	return b, nil
}

// InitializeFromProtoUnsafeAltair directly uses the beacon state protobuf fields
// and sets them as fields of the BeaconState type.
func InitializeFromProtoUnsafeAltair(st *ethpb.BeaconStateAltair) (state.BeaconStateAltair, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	var bRoots customtypes.BlockRoots
	for i, r := range st.BlockRoots {
		bRoots[i] = bytesutil.ToBytes32(r)
	}
	var sRoots customtypes.StateRoots
	for i, r := range st.StateRoots {
		sRoots[i] = bytesutil.ToBytes32(r)
	}
	hRoots := customtypes.HistoricalRoots(make([][32]byte, len(st.HistoricalRoots)))
	for i, r := range st.HistoricalRoots {
		hRoots[i] = bytesutil.ToBytes32(r)
	}
	var mixes customtypes.RandaoMixes
	for i, m := range st.RandaoMixes {
		mixes[i] = bytesutil.ToBytes32(m)
	}

	fieldCount := params.BeaconConfig().BeaconStateAltairFieldCount
	b := &BeaconState{
		version:                     Altair,
		genesisTime:                 st.GenesisTime,
		genesisValidatorsRoot:       bytesutil.ToBytes32(st.GenesisValidatorsRoot),
		slot:                        st.Slot,
		fork:                        st.Fork,
		latestBlockHeader:           st.LatestBlockHeader,
		blockRoots:                  &bRoots,
		stateRoots:                  &sRoots,
		historicalRoots:             hRoots,
		eth1Data:                    st.Eth1Data,
		eth1DataVotes:               st.Eth1DataVotes,
		eth1DepositIndex:            st.Eth1DepositIndex,
		validators:                  st.Validators,
		balances:                    st.Balances,
		randaoMixes:                 &mixes,
		slashings:                   st.Slashings,
		previousEpochParticipation:  st.PreviousEpochParticipation,
		currentEpochParticipation:   st.CurrentEpochParticipation,
		justificationBits:           st.JustificationBits,
		previousJustifiedCheckpoint: st.PreviousJustifiedCheckpoint,
		currentJustifiedCheckpoint:  st.CurrentJustifiedCheckpoint,
		finalizedCheckpoint:         st.FinalizedCheckpoint,
		inactivityScores:            st.InactivityScores,
		currentSyncCommittee:        st.CurrentSyncCommittee,
		nextSyncCommittee:           st.NextSyncCommittee,

		dirtyFields:           make(map[int]bool, fieldCount),
		dirtyIndices:          make(map[int][]uint64, fieldCount),
		stateFieldLeaves:      make(map[int]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[int]*stateutil.Reference, 11),
		rebuildTrie:           make(map[int]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	b.fieldIndexes = make(map[int]v0types.FieldIndex, fieldCount)
	b.fieldIndexes[0] = v0types.GenesisTime
	b.fieldIndexes[1] = v0types.GenesisValidatorsRoot
	b.fieldIndexes[2] = v0types.Slot
	b.fieldIndexes[3] = v0types.Fork
	b.fieldIndexes[4] = v0types.LatestBlockHeader
	b.fieldIndexes[5] = v0types.BlockRoots
	b.fieldIndexes[6] = v0types.StateRoots
	b.fieldIndexes[7] = v0types.HistoricalRoots
	b.fieldIndexes[8] = v0types.Eth1Data
	b.fieldIndexes[9] = v0types.Eth1DataVotes
	b.fieldIndexes[10] = v0types.Eth1DepositIndex
	b.fieldIndexes[11] = v0types.Validators
	b.fieldIndexes[12] = v0types.Balances
	b.fieldIndexes[13] = v0types.RandaoMixes
	b.fieldIndexes[14] = v0types.Slashings
	b.fieldIndexes[15] = v0types.PreviousEpochParticipationBits
	b.fieldIndexes[16] = v0types.CurrentEpochParticipationBits
	b.fieldIndexes[17] = v0types.JustificationBits
	b.fieldIndexes[18] = v0types.PreviousJustifiedCheckpoint
	b.fieldIndexes[19] = v0types.CurrentJustifiedCheckpoint
	b.fieldIndexes[20] = v0types.FinalizedCheckpoint
	b.fieldIndexes[21] = v0types.InactivityScores
	b.fieldIndexes[22] = v0types.CurrentSyncCommittee
	b.fieldIndexes[23] = v0types.NextSyncCommittee
	b.fieldIndexesRev = make(map[v0types.FieldIndex]int, fieldCount)
	b.fieldIndexesRev[v0types.GenesisTime] = 0
	b.fieldIndexesRev[v0types.GenesisValidatorsRoot] = 1
	b.fieldIndexesRev[v0types.Slot] = 2
	b.fieldIndexesRev[v0types.Fork] = 3
	b.fieldIndexesRev[v0types.LatestBlockHeader] = 4
	b.fieldIndexesRev[v0types.BlockRoots] = 5
	b.fieldIndexesRev[v0types.StateRoots] = 6
	b.fieldIndexesRev[v0types.HistoricalRoots] = 7
	b.fieldIndexesRev[v0types.Eth1Data] = 8
	b.fieldIndexesRev[v0types.Eth1DataVotes] = 9
	b.fieldIndexesRev[v0types.Eth1DepositIndex] = 10
	b.fieldIndexesRev[v0types.Validators] = 11
	b.fieldIndexesRev[v0types.Balances] = 12
	b.fieldIndexesRev[v0types.RandaoMixes] = 13
	b.fieldIndexesRev[v0types.Slashings] = 14
	b.fieldIndexesRev[v0types.PreviousEpochParticipationBits] = 15
	b.fieldIndexesRev[v0types.CurrentEpochParticipationBits] = 16
	b.fieldIndexesRev[v0types.JustificationBits] = 17
	b.fieldIndexesRev[v0types.PreviousJustifiedCheckpoint] = 18
	b.fieldIndexesRev[v0types.CurrentJustifiedCheckpoint] = 19
	b.fieldIndexesRev[v0types.FinalizedCheckpoint] = 20
	b.fieldIndexesRev[v0types.InactivityScores] = 21
	b.fieldIndexesRev[v0types.CurrentSyncCommittee] = 22
	b.fieldIndexesRev[v0types.NextSyncCommittee] = 23

	var err error
	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[i] = true
		b.rebuildTrie[i] = true
		b.dirtyIndices[i] = []uint64{}
		b.stateFieldLeaves[i], err = fieldtrie.NewFieldTrie(v0types.FieldIndex(i), types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.BlockRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.StateRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.HistoricalRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Eth1DataVotes]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Validators]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Balances]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.RandaoMixes]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Slashings]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochAttestations]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochAttestations]] = stateutil.NewRef(1)

	state.StateCount.Inc()
	return b, nil
}

// InitializeFromProtoUnsafeBellatrix directly uses the beacon state protobuf fields
// and sets them as fields of the BeaconState type.
func InitializeFromProtoUnsafeBellatrix(st *ethpb.BeaconStateBellatrix) (state.BeaconStateBellatrix, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	var bRoots customtypes.BlockRoots
	for i, r := range st.BlockRoots {
		bRoots[i] = bytesutil.ToBytes32(r)
	}
	var sRoots customtypes.StateRoots
	for i, r := range st.StateRoots {
		sRoots[i] = bytesutil.ToBytes32(r)
	}
	hRoots := customtypes.HistoricalRoots(make([][32]byte, len(st.HistoricalRoots)))
	for i, r := range st.HistoricalRoots {
		hRoots[i] = bytesutil.ToBytes32(r)
	}
	var mixes customtypes.RandaoMixes
	for i, m := range st.RandaoMixes {
		mixes[i] = bytesutil.ToBytes32(m)
	}

	fieldCount := params.BeaconConfig().BeaconStateBellatrixFieldCount
	b := &BeaconState{
		version:                      Bellatrix,
		genesisTime:                  st.GenesisTime,
		genesisValidatorsRoot:        bytesutil.ToBytes32(st.GenesisValidatorsRoot),
		slot:                         st.Slot,
		fork:                         st.Fork,
		latestBlockHeader:            st.LatestBlockHeader,
		blockRoots:                   &bRoots,
		stateRoots:                   &sRoots,
		historicalRoots:              hRoots,
		eth1Data:                     st.Eth1Data,
		eth1DataVotes:                st.Eth1DataVotes,
		eth1DepositIndex:             st.Eth1DepositIndex,
		validators:                   st.Validators,
		balances:                     st.Balances,
		randaoMixes:                  &mixes,
		slashings:                    st.Slashings,
		previousEpochParticipation:   st.PreviousEpochParticipation,
		currentEpochParticipation:    st.CurrentEpochParticipation,
		justificationBits:            st.JustificationBits,
		previousJustifiedCheckpoint:  st.PreviousJustifiedCheckpoint,
		currentJustifiedCheckpoint:   st.CurrentJustifiedCheckpoint,
		finalizedCheckpoint:          st.FinalizedCheckpoint,
		inactivityScores:             st.InactivityScores,
		currentSyncCommittee:         st.CurrentSyncCommittee,
		nextSyncCommittee:            st.NextSyncCommittee,
		latestExecutionPayloadHeader: st.LatestExecutionPayloadHeader,

		dirtyFields:           make(map[int]bool, fieldCount),
		dirtyIndices:          make(map[int][]uint64, fieldCount),
		stateFieldLeaves:      make(map[int]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[int]*stateutil.Reference, 11),
		rebuildTrie:           make(map[int]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	b.fieldIndexes = make(map[int]v0types.FieldIndex, fieldCount)
	b.fieldIndexes[0] = v0types.GenesisTime
	b.fieldIndexes[1] = v0types.GenesisValidatorsRoot
	b.fieldIndexes[2] = v0types.Slot
	b.fieldIndexes[3] = v0types.Fork
	b.fieldIndexes[4] = v0types.LatestBlockHeader
	b.fieldIndexes[5] = v0types.BlockRoots
	b.fieldIndexes[6] = v0types.StateRoots
	b.fieldIndexes[7] = v0types.HistoricalRoots
	b.fieldIndexes[8] = v0types.Eth1Data
	b.fieldIndexes[9] = v0types.Eth1DataVotes
	b.fieldIndexes[10] = v0types.Eth1DepositIndex
	b.fieldIndexes[11] = v0types.Validators
	b.fieldIndexes[12] = v0types.Balances
	b.fieldIndexes[13] = v0types.RandaoMixes
	b.fieldIndexes[14] = v0types.Slashings
	b.fieldIndexes[15] = v0types.PreviousEpochParticipationBits
	b.fieldIndexes[16] = v0types.CurrentEpochParticipationBits
	b.fieldIndexes[17] = v0types.JustificationBits
	b.fieldIndexes[18] = v0types.PreviousJustifiedCheckpoint
	b.fieldIndexes[19] = v0types.CurrentJustifiedCheckpoint
	b.fieldIndexes[20] = v0types.FinalizedCheckpoint
	b.fieldIndexes[21] = v0types.InactivityScores
	b.fieldIndexes[22] = v0types.CurrentSyncCommittee
	b.fieldIndexes[23] = v0types.NextSyncCommittee
	b.fieldIndexes[24] = v0types.LatestExecutionPayloadHeader
	b.fieldIndexesRev = make(map[v0types.FieldIndex]int, fieldCount)
	b.fieldIndexesRev[v0types.GenesisTime] = 0
	b.fieldIndexesRev[v0types.GenesisValidatorsRoot] = 1
	b.fieldIndexesRev[v0types.Slot] = 2
	b.fieldIndexesRev[v0types.Fork] = 3
	b.fieldIndexesRev[v0types.LatestBlockHeader] = 4
	b.fieldIndexesRev[v0types.BlockRoots] = 5
	b.fieldIndexesRev[v0types.StateRoots] = 6
	b.fieldIndexesRev[v0types.HistoricalRoots] = 7
	b.fieldIndexesRev[v0types.Eth1Data] = 8
	b.fieldIndexesRev[v0types.Eth1DataVotes] = 9
	b.fieldIndexesRev[v0types.Eth1DepositIndex] = 10
	b.fieldIndexesRev[v0types.Validators] = 11
	b.fieldIndexesRev[v0types.Balances] = 12
	b.fieldIndexesRev[v0types.RandaoMixes] = 13
	b.fieldIndexesRev[v0types.Slashings] = 14
	b.fieldIndexesRev[v0types.PreviousEpochParticipationBits] = 15
	b.fieldIndexesRev[v0types.CurrentEpochParticipationBits] = 16
	b.fieldIndexesRev[v0types.JustificationBits] = 17
	b.fieldIndexesRev[v0types.PreviousJustifiedCheckpoint] = 18
	b.fieldIndexesRev[v0types.CurrentJustifiedCheckpoint] = 19
	b.fieldIndexesRev[v0types.FinalizedCheckpoint] = 20
	b.fieldIndexesRev[v0types.InactivityScores] = 21
	b.fieldIndexesRev[v0types.CurrentSyncCommittee] = 22
	b.fieldIndexesRev[v0types.NextSyncCommittee] = 23
	b.fieldIndexesRev[v0types.LatestExecutionPayloadHeader] = 24

	var err error
	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[i] = true
		b.rebuildTrie[i] = true
		b.dirtyIndices[i] = []uint64{}
		b.stateFieldLeaves[i], err = fieldtrie.NewFieldTrie(v0types.FieldIndex(i), types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.BlockRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.StateRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.HistoricalRoots]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Eth1DataVotes]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Validators]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Balances]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.RandaoMixes]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.Slashings]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochAttestations]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochAttestations]] = stateutil.NewRef(1)
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.LatestExecutionPayloadHeader]] = stateutil.NewRef(1)

	state.StateCount.Inc()
	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() state.BeaconState {
	b.lock.RLock()
	defer b.lock.RUnlock()
	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	dst := &BeaconState{
		// Primitive v0types, safe to copy.
		genesisTime:      b.genesisTime,
		slot:             b.slot,
		eth1DepositIndex: b.eth1DepositIndex,

		// Large arrays, infrequently changed, constant size.
		blockRoots:                b.blockRoots,
		stateRoots:                b.stateRoots,
		randaoMixes:               b.randaoMixes,
		previousEpochAttestations: b.previousEpochAttestations,
		currentEpochAttestations:  b.currentEpochAttestations,
		eth1DataVotes:             b.eth1DataVotes,
		slashings:                 b.slashings,

		// Large arrays, increases over time.
		balances:                   b.balances,
		historicalRoots:            b.historicalRoots,
		validators:                 b.validators,
		previousEpochParticipation: b.previousEpochParticipation,
		currentEpochParticipation:  b.currentEpochParticipation,
		inactivityScores:           b.inactivityScores,

		// Everything else, too small to be concerned about, constant size.
		genesisValidatorsRoot:        b.genesisValidatorsRoot,
		justificationBits:            b.justificationBitsVal(),
		fork:                         b.forkVal(),
		latestBlockHeader:            b.latestBlockHeaderVal(),
		eth1Data:                     b.eth1DataVal(),
		previousJustifiedCheckpoint:  b.previousJustifiedCheckpointVal(),
		currentJustifiedCheckpoint:   b.currentJustifiedCheckpointVal(),
		finalizedCheckpoint:          b.finalizedCheckpointVal(),
		currentSyncCommittee:         b.currentSyncCommitteeVal(),
		nextSyncCommittee:            b.nextSyncCommitteeVal(),
		latestExecutionPayloadHeader: b.latestExecutionPayloadHeaderVal(),

		dirtyFields:           make(map[int]bool, fieldCount),
		dirtyIndices:          make(map[int][]uint64, fieldCount),
		rebuildTrie:           make(map[int]bool, fieldCount),
		sharedFieldReferences: make(map[int]*stateutil.Reference, 10),
		stateFieldLeaves:      make(map[int]*fieldtrie.FieldTrie, fieldCount),

		// Share the reference to validator index map.
		valMapHandler: b.valMapHandler,
	}

	for field, ref := range b.sharedFieldReferences {
		ref.AddRef()
		dst.sharedFieldReferences[field] = ref
	}

	// Increment ref for validator map
	b.valMapHandler.AddRef()

	for i := range b.dirtyFields {
		dst.dirtyFields[i] = true
	}

	for i := range b.dirtyIndices {
		indices := make([]uint64, len(b.dirtyIndices[i]))
		copy(indices, b.dirtyIndices[i])
		dst.dirtyIndices[i] = indices
	}

	for i := range b.rebuildTrie {
		dst.rebuildTrie[i] = true
	}

	for fldIdx, fieldTrie := range b.stateFieldLeaves {
		dst.stateFieldLeaves[fldIdx] = fieldTrie
		if fieldTrie.FieldReference() != nil {
			fieldTrie.Lock()
			fieldTrie.FieldReference().AddRef()
			fieldTrie.Unlock()
		}
	}

	if b.merkleLayers != nil {
		dst.merkleLayers = make([][][]byte, len(b.merkleLayers))
		for i, layer := range b.merkleLayers {
			dst.merkleLayers[i] = make([][]byte, len(layer))
			for j, content := range layer {
				dst.merkleLayers[i][j] = make([]byte, len(content))
				copy(dst.merkleLayers[i][j], content)
			}
		}
	}

	state.StateCount.Inc()
	// Finalizer runs when dst is being destroyed in garbage collection.
	runtime.SetFinalizer(dst, func(b *BeaconState) {
		for field, v := range b.sharedFieldReferences {
			v.MinusRef()
			if b.stateFieldLeaves[field].FieldReference() != nil {
				b.stateFieldLeaves[field].FieldReference().MinusRef()
			}

		}
		for i := 0; i < fieldCount; i++ {
			delete(b.stateFieldLeaves, i)
			delete(b.dirtyIndices, i)
			delete(b.dirtyFields, i)
			delete(b.sharedFieldReferences, i)
			delete(b.stateFieldLeaves, i)
		}
		state.StateCount.Sub(1)
	})
	return dst
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the Ethereum Simple Serialize specification.
func (b *BeaconState) HashTreeRoot(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.HashTreeRoot")
	defer span.End()

	b.lock.Lock()
	defer b.lock.Unlock()
	if err := b.initializeMerkleLayers(ctx); err != nil {
		return [32]byte{}, err
	}
	if err := b.recomputeDirtyFields(ctx); err != nil {
		return [32]byte{}, err
	}
	return bytesutil.ToBytes32(b.merkleLayers[len(b.merkleLayers)-1][0]), nil
}

// Initializes the Merkle layers for the beacon state if they are empty.
// WARNING: Caller must acquire the mutex before using.
func (b *BeaconState) initializeMerkleLayers(ctx context.Context) error {
	if len(b.merkleLayers) > 0 {
		return nil
	}
	fieldRoots, err := ComputeFieldRootsWithHasher(ctx, b)
	if err != nil {
		return err
	}
	layers := stateutil.Merkleize(fieldRoots)
	b.merkleLayers = layers
	switch b.version {
	case Phase0:
		b.dirtyFields = make(map[int]bool, params.BeaconConfig().BeaconStateFieldCount)
	case Altair:
		b.dirtyFields = make(map[int]bool, params.BeaconConfig().BeaconStateAltairFieldCount)
	case Bellatrix:
		b.dirtyFields = make(map[int]bool, params.BeaconConfig().BeaconStateBellatrixFieldCount)
	}

	return nil
}

// Recomputes the Merkle layers for the dirty fields in the state.
// WARNING: Caller must acquire the mutex before using.
func (b *BeaconState) recomputeDirtyFields(ctx context.Context) error {
	for field := range b.dirtyFields {
		root, err := b.rootSelector(ctx, field)
		if err != nil {
			return err
		}
		b.merkleLayers[0][field] = root[:]
		b.recomputeRoot(field)
		delete(b.dirtyFields, field)
	}
	return nil
}

// FieldReferencesCount returns the reference count held by each field. This
// also includes the field trie held by each field.
func (b *BeaconState) FieldReferencesCount() map[string]uint64 {
	refMap := make(map[string]uint64)
	b.lock.RLock()
	defer b.lock.RUnlock()
	for i, f := range b.sharedFieldReferences {
		refMap[b.fieldIndexes[i].String()] = uint64(f.Refs())
	}
	for i, f := range b.stateFieldLeaves {
		numOfRefs := uint64(f.FieldReference().Refs())
		f.RLock()
		if !f.Empty() {
			refMap[b.fieldIndexes[i].String()+"_trie"] = numOfRefs
		}
		f.RUnlock()
	}
	return refMap
}

// IsNil checks if the state and the underlying proto
// object are nil.
func (b *BeaconState) IsNil() bool {
	return b == nil
}

func (b *BeaconState) rootSelector(ctx context.Context, field int) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "beaconState.rootSelector")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("field", b.fieldIndexes[field].String()))

	hasher := hash.CustomSHA256Hasher()
	switch b.fieldIndexes[field] {
	case v0types.GenesisTime:
		return ssz.Uint64Root(b.genesisTime), nil
	case v0types.GenesisValidatorsRoot:
		return b.genesisValidatorsRoot, nil
	case v0types.Slot:
		return ssz.Uint64Root(uint64(b.slot)), nil
	case v0types.Eth1DepositIndex:
		return ssz.Uint64Root(b.eth1DepositIndex), nil
	case v0types.Fork:
		return ssz.ForkRoot(b.fork)
	case v0types.LatestBlockHeader:
		return stateutil.BlockHeaderRoot(b.latestBlockHeader)
	case v0types.BlockRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.blockRoots, fieldparams.BlockRootsLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.blockRoots)
	case v0types.StateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.stateRoots, fieldparams.StateRootsLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.stateRoots)
	case v0types.HistoricalRoots:
		hRoots := make([][]byte, len(b.historicalRoots))
		for i := range hRoots {
			hRoots[i] = b.historicalRoots[i][:]
		}
		return ssz.ByteArrayRootWithLimit(hRoots, fieldparams.HistoricalRootsLength)
	case v0types.Eth1Data:
		return stateutil.Eth1Root(hasher, b.eth1Data)
	case v0types.Eth1DataVotes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.eth1DataVotes,
				fieldparams.Eth1DataVotesLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.eth1DataVotes)
	case v0types.Validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.validators, fieldparams.ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(11, b.validators)
	case v0types.Balances:
		if features.Get().EnableBalanceTrieComputation {
			if b.rebuildTrie[field] {
				maxBalCap := uint64(fieldparams.ValidatorRegistryLimit)
				elemSize := uint64(8)
				balLimit := (maxBalCap*elemSize + 31) / 32
				err := b.resetFieldTrie(field, b.balances, balLimit)
				if err != nil {
					return [32]byte{}, err
				}
				delete(b.rebuildTrie, field)
				return b.stateFieldLeaves[field].TrieRoot()
			}
			return b.recomputeFieldTrie(12, b.balances)
		}
		return stateutil.Uint64ListRootWithRegistryLimit(b.balances)
	case v0types.RandaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.randaoMixes, fieldparams.RandaoMixesLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(13, b.randaoMixes)
	case v0types.Slashings:
		return ssz.SlashingsRoot(b.slashings)
	case v0types.PreviousEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.previousEpochAttestations,
				fieldparams.PreviousEpochAttestationsLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.previousEpochAttestations)
	case v0types.CurrentEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.currentEpochAttestations,
				fieldparams.CurrentEpochAttestationsLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.currentEpochAttestations)
	case v0types.PreviousEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.previousEpochParticipation)
	case v0types.CurrentEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.currentEpochParticipation)
	case v0types.JustificationBits:
		return bytesutil.ToBytes32(b.justificationBits), nil
	case v0types.PreviousJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.previousJustifiedCheckpoint)
	case v0types.CurrentJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.currentJustifiedCheckpoint)
	case v0types.FinalizedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.finalizedCheckpoint)
	case v0types.InactivityScores:
		return stateutil.Uint64ListRootWithRegistryLimit(b.inactivityScores)
	case v0types.CurrentSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.currentSyncCommittee)
	case v0types.NextSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.nextSyncCommittee)
	case v0types.LatestExecutionPayloadHeader:
		return b.latestExecutionPayloadHeader.HashTreeRoot()
	}
	return [32]byte{}, errors.New("invalid field index provided")
}

func (b *BeaconState) recomputeFieldTrie(index int, elements interface{}) ([32]byte, error) {
	fTrie := b.stateFieldLeaves[index]
	// We can't lock the trie directly because the trie's variable gets reassigned,
	// and therefore we would call Unlock() on a different object.
	fTrieMutex := fTrie.RWMutex
	if fTrie.FieldReference().Refs() > 1 {
		fTrieMutex.Lock()
		fTrie.FieldReference().MinusRef()
		newTrie := fTrie.CopyTrie()
		b.stateFieldLeaves[index] = newTrie
		fTrie = newTrie
		fTrieMutex.Unlock()
	}
	// remove duplicate indexes
	b.dirtyIndices[index] = slice.SetUint64(b.dirtyIndices[index])
	// sort indexes again
	sort.Slice(b.dirtyIndices[index], func(i int, j int) bool {
		return b.dirtyIndices[index][i] < b.dirtyIndices[index][j]
	})
	root, err := fTrie.RecomputeTrie(b.dirtyIndices[index], elements)
	if err != nil {
		return [32]byte{}, err
	}
	b.dirtyIndices[index] = []uint64{}
	return root, nil
}

func (b *BeaconState) resetFieldTrie(index int, elements interface{}, length uint64) error {
	fTrie, err := fieldtrie.NewFieldTrie(b.fieldIndexes[index], fieldMap[b.fieldIndexes[index]], elements, length)
	if err != nil {
		return err
	}
	b.stateFieldLeaves[index] = fTrie
	b.dirtyIndices[index] = []uint64{}
	return nil
}

// TODO: Better doc
// ComputeFieldRootsWithHasher hashes the provided state and returns its respective field roots.
func ComputeFieldRootsWithHasher(ctx context.Context, state *BeaconState) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasher")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	var fieldRoots [][]byte
	switch state.Version() {
	case int(Phase0):
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateFieldCount)
	case int(Altair):
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateAltairFieldCount)
	case int(Bellatrix):
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateBellatrixFieldCount)
	}

	fieldRootIx := 0

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.GenesisTime())
	fieldRoots[fieldRootIx] = genesisRoot[:]
	fieldRootIx++

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.GenesisValidatorsRoot())
	fieldRoots[fieldRootIx] = r[:]
	fieldRootIx++

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.Slot()))
	fieldRoots[fieldRootIx] = slotRoot[:]
	fieldRootIx++

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.Fork())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[fieldRootIx] = forkHashTreeRoot[:]
	fieldRootIx++

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[fieldRootIx] = headerHashTreeRoot[:]
	fieldRootIx++

	// BlockRoots array root.
	blockRootsRoot, err := stateutil.ArraysRoot(state.BlockRoots(), fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[fieldRootIx] = blockRootsRoot[:]
	fieldRootIx++

	// StateRoots array root.
	stateRootsRoot, err := stateutil.ArraysRoot(state.StateRoots(), fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[fieldRootIx] = stateRootsRoot[:]
	fieldRootIx++

	// HistoricalRoots slice root.
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(state.HistoricalRoots(), fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[fieldRootIx] = historicalRootsRt[:]
	fieldRootIx++

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := stateutil.Eth1Root(hasher, state.Eth1Data())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[fieldRootIx] = eth1HashTreeRoot[:]
	fieldRootIx++

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := stateutil.Eth1DataVotesRoot(state.Eth1DataVotes())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[fieldRootIx] = eth1VotesRoot[:]
	fieldRootIx++

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex())
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[fieldRootIx] = eth1DepositBuf[:]
	fieldRootIx++

	// Validators slice root.
	validatorsRoot, err := stateutil.ValidatorRegistryRoot(state.Validators())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[fieldRootIx] = validatorsRoot[:]
	fieldRootIx++

	// Balances slice root.
	balancesRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.Balances())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[fieldRootIx] = balancesRoot[:]
	fieldRootIx++

	// RandaoMixes array root.
	randaoRootsRoot, err := stateutil.ArraysRoot(state.RandaoMixes(), fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[fieldRootIx] = randaoRootsRoot[:]
	fieldRootIx++

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.Slashings())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[fieldRootIx] = slashingsRootsRoot[:]
	fieldRootIx++

	if state.Version() == version.Phase0 {
		// PreviousEpochAttestations slice root.
		prevEpochAtts, err := state.PreviousEpochAttestations()
		if err != nil {
			return nil, errors.Wrap(err, "could not get previous epoch attestations")
		}
		prevAttsRoot, err := stateutil.EpochAttestationsRoot(prevEpochAtts)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = prevAttsRoot[:]
		fieldRootIx++

		// CurrentEpochAttestations slice root.
		currEpochAtts, err := state.CurrentEpochAttestations()
		if err != nil {
			return nil, errors.Wrap(err, "could not get current epoch attestations")
		}
		currAttsRoot, err := stateutil.EpochAttestationsRoot(currEpochAtts)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = currAttsRoot[:]
		fieldRootIx++
	}

	if state.Version() == version.Altair || state.Version() == version.Bellatrix {
		// PreviousEpochParticipation slice root.
		prevEpochParticipation, err := state.PreviousEpochParticipation()
		if err != nil {
			return nil, errors.Wrap(err, "could not get previous epoch participation")
		}
		prevParticipationRoot, err := stateutil.ParticipationBitsRoot(prevEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = prevParticipationRoot[:]
		fieldRootIx++

		// CurrentEpochParticipation slice root.
		currEpochParticipation, err := state.CurrentEpochParticipation()
		if err != nil {
			return nil, errors.Wrap(err, "could not get current epoch participation")
		}
		currParticipationRoot, err := stateutil.ParticipationBitsRoot(currEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = currParticipationRoot[:]
		fieldRootIx++
	}

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits())
	fieldRoots[fieldRootIx] = justifiedBitsRoot[:]
	fieldRootIx++

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = prevCheckRoot[:]
	fieldRootIx++

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = currJustRoot[:]
	fieldRootIx++

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.FinalizedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = finalRoot[:]
	fieldRootIx++

	if state.Version() == version.Altair || state.Version() == version.Bellatrix {
		// Current sync committee root.
		currSyncCommittee, err := state.CurrentSyncCommittee()
		if err != nil {
			return nil, errors.Wrap(err, "could not get current sync committee")
		}
		currentSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(currSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = currentSyncCommitteeRoot[:]
		fieldRootIx++

		// Next sync committee root.
		nextSyncCommittee, err := state.NextSyncCommittee()
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee")
		}
		nextSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(nextSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = nextSyncCommitteeRoot[:]
		fieldRootIx++
	}

	if state.Version() == version.Bellatrix {
		// Execution payload root.
		header, err := state.LatestExecutionPayloadHeader()
		if err != nil {
			return nil, errors.Wrap(err, "could not get latest execution payload header")
		}
		executionPayloadRoot, err := header.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[fieldRootIx] = executionPayloadRoot[:]
	}

	return fieldRoots, nil
}
