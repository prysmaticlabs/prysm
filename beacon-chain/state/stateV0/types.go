package stateV0

import (
	"sync"

	"github.com/pkg/errors"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Ensure type BeaconState below implements BeaconState interface.
var _ iface.BeaconState = (*BeaconState)(nil)

// ErrNilInnerState returns when the inner state is nil and no copy set or get
// operations can be performed on state.
var ErrNilInnerState = errors.New("nil inner state")

// BeaconState defines a struct containing utilities for the eth2 chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	state                 *pbp2p.BeaconState
	lock                  sync.RWMutex
	dirtyFields           map[stateutil.FieldIndex]interface{}
	dirtyIndices          map[stateutil.FieldIndex][]uint64
	stateFieldLeaves      map[stateutil.FieldIndex]*stateutil.FieldTrie
	rebuildTrie           map[stateutil.FieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[stateutil.FieldIndex]*stateutil.Reference
}
