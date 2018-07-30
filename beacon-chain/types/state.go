package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"golang.org/x/crypto/blake2b"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
// TODO: Change ActiveState to use proto
type ActiveState struct {
	TotalAttesterDeposits uint64 // TotalAttesterDeposits is the total quantity of wei that attested for the most recent checkpoint.
	AttesterBitfields     []byte // AttesterBitfields represents which validator has attested.
}

// CrystallizedState contains fields of every epoch state,
// it changes every epoch.
type CrystallizedState struct {
	data *pb.CrystallizedStateResponse
}

// NewCrystallizedState creates a new crystallized state given certain epoch.
func NewCrystallizedState(epoch uint64) *CrystallizedState {
	data := &pb.CrystallizedStateResponse{CurrentEpoch: epoch}
	return &CrystallizedState{data: data}
}

// NewCrystallizedStateWithData explicitly sets the data field of a crystallized state.
func NewCrystallizedStateWithData(data *pb.CrystallizedStateResponse) *CrystallizedState {
	return &CrystallizedState{data: data}
}

// NewGenesisStates initializes a beacon chain with starting parameters.
func NewGenesisStates() (*ActiveState, *CrystallizedState) {
	active := &ActiveState{
		TotalAttesterDeposits: 0,
		AttesterBitfields:     []byte{},
	}
	crystallized := &CrystallizedState{
		data: &pb.CrystallizedStateResponse{
			ActiveValidators:      []*pb.ValidatorRecord{},
			QueuedValidators:      []*pb.ValidatorRecord{},
			ExitedValidators:      []*pb.ValidatorRecord{},
			CurrentEpochShuffling: []uint64{},
			CurrentEpoch:          0,
			LastJustifiedEpoch:    0,
			LastFinalizedEpoch:    0,
			CurrentDynasty:        0,
			TotalDeposits:         0,
			DynastySeed:           []byte{},
			DynastySeedLastReset:  0,
		},
	}
	return active, crystallized
}

// Marshal encodes crystallized state object into the wire format.
func (c *CrystallizedState) Marshal() ([]byte, error) {
	return proto.Marshal(c.data)
}

// Hash serializes the crystallized state object then uses
// blake2b to hash the serialized object.
func (c *CrystallizedState) Hash() ([32]byte, error) {
	data, err := proto.Marshal(c.data)
	if err != nil {
		return [32]byte{}, err
	}
	return blake2b.Sum256(data), nil
}

// ActiveValidators returns list of validator that are active.
func (c *CrystallizedState) ActiveValidators() []*pb.ValidatorRecord {
	return c.data.ActiveValidators
}

// ActiveValidatorsLength returns the number of total active validators.
func (c *CrystallizedState) ActiveValidatorsLength() int {
	return len(c.data.ActiveValidators)
}

// UpdateActiveValidators updates active validator set.
func (c *CrystallizedState) UpdateActiveValidators(validators []*pb.ValidatorRecord) {
	c.data.ActiveValidators = validators
}

// QueuedValidators returns list of validator that are queued.
func (c *CrystallizedState) QueuedValidators() []*pb.ValidatorRecord {
	return c.data.QueuedValidators
}

// QueuedValidatorsLength returns the number of total queued validators.
func (c *CrystallizedState) QueuedValidatorsLength() int {
	return len(c.data.QueuedValidators)
}

// UpdateQueuedValidators updates queued validator set.
func (c *CrystallizedState) UpdateQueuedValidators(validators []*pb.ValidatorRecord) {
	c.data.QueuedValidators = validators
}

// ExitedValidators returns list of validator that have exited.
func (c *CrystallizedState) ExitedValidators() []*pb.ValidatorRecord {
	return c.data.ExitedValidators
}

// ExitedValidatorsLength returns the number of total exited validators.
func (c *CrystallizedState) ExitedValidatorsLength() int {
	return len(c.data.ExitedValidators)
}

// UpdateExitedValidators updates active validator set.
func (c *CrystallizedState) UpdateExitedValidators(validators []*pb.ValidatorRecord) {
	c.data.ExitedValidators = validators
}

// CurrentEpochShuffling is the permutation of validators that determines
// who participates in what committee and at what height.
func (c *CrystallizedState) CurrentEpochShuffling() []uint64 {
	return c.data.CurrentEpochShuffling
}

// CurrentEpoch of the beacon chain.
func (c *CrystallizedState) CurrentEpoch() uint64 {
	return c.data.CurrentEpoch
}

// LastJustifiedEpoch of the beacon chain.
func (c *CrystallizedState) LastJustifiedEpoch() uint64 {
	return c.data.LastJustifiedEpoch
}

// SetLastJustifiedEpoch sets last justified epoch of the beacon chain.
func (c *CrystallizedState) SetLastJustifiedEpoch(epoch uint64) {
	c.data.LastJustifiedEpoch = epoch
}

// LastFinalizedEpoch of the beacon chain.
func (c *CrystallizedState) LastFinalizedEpoch() uint64 {
	return c.data.LastFinalizedEpoch
}

// SetLastFinalizedEpoch sets last justified epoch of the beacon chain.
func (c *CrystallizedState) SetLastFinalizedEpoch(epoch uint64) {
	c.data.LastFinalizedEpoch = epoch
}

// CurrentDynasty of the beacon chain.
func (c *CrystallizedState) CurrentDynasty() uint64 {
	return c.data.CurrentDynasty
}

// NextShard crosslink assignment will be coming from.
func (c *CrystallizedState) NextShard() uint64 {
	return c.data.NextShard
}

// CurrentCheckPoint for the FFG state.
func (c *CrystallizedState) CurrentCheckPoint() common.Hash {
	return common.BytesToHash(c.data.CurrentCheckPoint)
}

// TotalDeposits is combined deposits of all the validators.
func (c *CrystallizedState) TotalDeposits() uint64 {
	return c.data.TotalDeposits
}

// DynastySeed is used to select the committee for each shard.
func (c *CrystallizedState) DynastySeed() common.Hash {
	return common.BytesToHash(c.data.DynastySeed)
}

// DynastySeedLastReset is the last epoch the crosslink seed was reset.
func (c *CrystallizedState) DynastySeedLastReset() uint64 {
	return c.data.DynastySeedLastReset
}
