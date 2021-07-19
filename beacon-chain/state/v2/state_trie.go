package v2

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/protobuf/proto"
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *pbp2p.BeaconStateAltair) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*pbp2p.BeaconStateAltair))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *pbp2p.BeaconStateAltair) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	fieldCount := params.BeaconConfig().BeaconStateAltairFieldCount
	b := &BeaconState{
		state:                 st,
		dirtyFields:           make(map[fieldIndex]interface{}, fieldCount),
		dirtyIndices:          make(map[fieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[fieldIndex]*FieldTrie, fieldCount),
		sharedFieldReferences: make(map[fieldIndex]*stateutil.Reference, 11),
		rebuildTrie:           make(map[fieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[fieldIndex(i)] = true
		b.rebuildTrie[fieldIndex(i)] = true
		b.dirtyIndices[fieldIndex(i)] = []uint64{}
		b.stateFieldLeaves[fieldIndex(i)] = &FieldTrie{
			field:     fieldIndex(i),
			reference: stateutil.NewRef(1),
			RWMutex:   new(sync.RWMutex),
		}
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[randaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[blockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1) // New in Altair.
	b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)  // New in Altair.
	b.sharedFieldReferences[slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[inactivityScores] = stateutil.NewRef(1) // New in Altair.
	b.sharedFieldReferences[historicalRoots] = stateutil.NewRef(1)

	return b, nil
}
