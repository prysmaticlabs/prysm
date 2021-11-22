package v3

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

var (
	stateCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_state_merge_count",
		Help: "Count the number of active beacon state objects.",
	})
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *ethpb.BeaconStateMerge) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*ethpb.BeaconStateMerge))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *ethpb.BeaconStateMerge) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	fieldCount := params.BeaconConfig().BeaconStateAltairFieldCount
	b := &BeaconState{
		state:                 st,
		dirtyFields:           make(map[types.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[types.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[types.FieldIndex]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[types.FieldIndex]*stateutil.Reference, 11),
		rebuildTrie:           make(map[types.FieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	var err error
	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[types.FieldIndex(i)] = true
		b.rebuildTrie[types.FieldIndex(i)] = true
		b.dirtyIndices[types.FieldIndex(i)] = []uint64{}
		b.stateFieldLeaves[types.FieldIndex(i)], err = fieldtrie.NewFieldTrie(types.FieldIndex(i), types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
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

	stateCount.Inc()
	return b, nil
}
