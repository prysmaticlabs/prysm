package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"golang.org/x/crypto/blake2b"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
type ActiveState struct {
	data *pb.ActiveStateResponse
}

// CrystallizedState contains fields of every epoch state,
// it changes every epoch.
type CrystallizedState struct {
	data *pb.CrystallizedStateResponse
}

// NewCrystallizedState creates a new crystallized state with a explicitly set data field.
func NewCrystallizedState(data *pb.CrystallizedStateResponse) *CrystallizedState {
	return &CrystallizedState{data: data}
}

// NewActiveState creates a new active state with a explicitly set data field.
func NewActiveState(data *pb.ActiveStateResponse) *ActiveState {
	return &ActiveState{data: data}
}

// NewGenesisStates initializes a beacon chain with starting parameters.
func NewGenesisStates() (*ActiveState, *CrystallizedState) {
	active := &ActiveState{
		data: &pb.ActiveStateResponse{
			TotalAttesterDeposits: 0,
			AttesterBitfield:      []byte{},
		},
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

// Marshal encodes active state object into the wire format.
func (a *ActiveState) Marshal() ([]byte, error) {
	return proto.Marshal(a.data)
}

// Hash serializes the active state object then uses
// blake2b to hash the serialized object.
func (a *ActiveState) Hash() ([32]byte, error) {
	data, err := proto.Marshal(a.data)
	if err != nil {
		return [32]byte{}, err
	}
	return blake2b.Sum256(data), nil
}

// TotalAttesterDeposits returns total quantity of wei that attested for the most recent checkpoint.
func (a *ActiveState) TotalAttesterDeposits() uint64 {
	return a.data.TotalAttesterDeposits
}

// SetTotalAttesterDeposits sets total quantity of wei that attested for the most recent checkpoint.
func (a *ActiveState) SetTotalAttesterDeposits(deposit uint64) {
	a.data.TotalAttesterDeposits = deposit
}

// AttesterBitfield returns a bitfield for seeing which attester has attested.
func (a *ActiveState) AttesterBitfield() []byte {
	return a.data.AttesterBitfield
}

// SetAttesterBitfield sets attester bitfield.
func (a *ActiveState) SetAttesterBitfield(bitfield []byte) {
	a.data.AttesterBitfield = bitfield
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
