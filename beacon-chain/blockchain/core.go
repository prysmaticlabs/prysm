package blockchain

import (
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/geth-sharding/beacon-chain/types"
	log "github.com/sirupsen/logrus"
)

var stateLookupKey = "beacon-chain-state"

// BeaconChain represents the core PoS blockchain object containing
// both a crystallized and active state.
type BeaconChain struct {
	activeState       *types.ActiveState
	crystallizedState *types.CrystallizedState
	lock              sync.Mutex
	db                ethdb.Database
}

// NewBeaconChain initializes an instance using genesis state parameters if
// none provided.
func NewBeaconChain(db ethdb.Database) (*BeaconChain, error) {
	if _, err := db.Get([]byte(stateLookupKey)); err != nil {
		log.Errorf("State does not exist: %v", err)
	}
	return &BeaconChain{
		db: db,
	}, nil
}

// ActiveState exposes a getter to external services.
func (b *BeaconChain) ActiveState() *types.ActiveState {
	return b.activeState
}

// CrystallizedState exposes a getter to external services.
func (b *BeaconChain) CrystallizedState() *types.CrystallizedState {
	return b.crystallizedState
}

// MutateActiveState allows external services to modify a beacon chain object.
func (b *BeaconChain) MutateActiveState(activeState *types.ActiveState) error {
	defer b.lock.Unlock()
	b.lock.Lock()
	b.activeState = activeState
	return b.persist()
}

// MutateCrystallizedState allows external services to modify the crystallized state.
func (b *BeaconChain) MutateCrystallizedState(crystallizedState *types.CrystallizedState) error {
	defer b.lock.Unlock()
	b.lock.Lock()
	b.crystallizedState = crystallizedState
	return b.persist()
}

// persist stores the RLP encoding of the latest beacon chain state into the db.
func (b *BeaconChain) persist() error {
	enc, err := rlp.EncodeToBytes(b)
	if err != nil {
		return err
	}
	return b.db.Put([]byte(stateLookupKey), enc)
}
