package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
// TODO: Change ActiveState to use proto
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
			PendingAttestations: []*pb.AttestationRecord{},
			RecentBlockHashes:   [][]byte{},
		},
	}
	crystallized := &CrystallizedState{
		data: &pb.CrystallizedStateResponse{
			EpochNumber:            0, //done
			JustifiedStreak:        0, //done
			LastJustifiedEpoch:     0, //done
			LastFinalizedEpoch:     0, //done
			CurrentDynasty:         0, //done
			CrosslinkingStartShard: 0, //done
			CurrentCheckPoint:      []byte{},
			TotalDeposits:          0,                       //done
			DynastySeed:            []byte{},                //done
			DynastySeedLastReset:   0,                       //done
			Validators:             []*pb.ValidatorRecord{}, //done
			IndicesForHeights:      []*pb.ArrayShardAndIndices{},
		},
	}
	return active, crystallized
}

// Proto returns the underlying protobuf data within a state primitive.
func (a *ActiveState) Proto() *pb.ActiveStateResponse {
	return a.data
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

// PendingAttestations returns attestations that have not yet been processed.
func (a *ActiveState) PendingAttestations() []*pb.AttestationRecord {
	return a.data.PendingAttestations
}

// ClearPendingAttestations clears attestations that have not yet been processed.
func (a *ActiveState) ClearPendingAttestations() {
	a.data.PendingAttestations = []*pb.AttestationRecord{}
}

// RecentBlockHashes returns the most recent 64 block hashes.
func (a *ActiveState) RecentBlockHashes() []common.Hash {
	var blockhashes []common.Hash
	for _, hash := range a.data.RecentBlockHashes {
		blockhashes = append(blockhashes, common.BytesToHash(hash))
	}
	return blockhashes
}

// ClearRecentBlockHashes resets the most recent 64 block hashes.
func (a *ActiveState) ClearRecentBlockHashes() {
	a.data.RecentBlockHashes = [][]byte{}
}

// Proto returns the underlying protobuf data within a state primitive.
func (c *CrystallizedState) Proto() *pb.CrystallizedStateResponse {
	return c.data
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

// EpochNumber returns current epoch number.
func (c *CrystallizedState) EpochNumber() uint64 {
	return c.data.EpochNumber
}

// JustifiedStreak returns number of consecutive justified slots ending at head.
func (c *CrystallizedState) JustifiedStreak() uint64 {
	return c.data.JustifiedStreak
}

// ClearJustifiedStreak clears the number of consecutive justified slots.
func (c *CrystallizedState) ClearJustifiedStreak() {
	c.data.JustifiedStreak = 0
}

// CrosslinkingStartShard returns next shard that crosslinking assignment will start from.
func (c *CrystallizedState) CrosslinkingStartShard() uint64 {
	return c.data.CrosslinkingStartShard
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

// TotalDeposits returns total balance of deposits.
func (c *CrystallizedState) TotalDeposits() uint64 {
	return c.data.TotalDeposits
}

// SetTotalDeposits sets total balance of deposits.
func (c *CrystallizedState) SetTotalDeposits(total uint64) {
	c.data.TotalDeposits = total
}

// CurrentCheckPoint for the FFG state.
func (c *CrystallizedState) CurrentCheckPoint() common.Hash {
	return common.BytesToHash(c.data.CurrentCheckPoint)
}

// DynastySeed is used to select the committee for each shard.
func (c *CrystallizedState) DynastySeed() common.Hash {
	return common.BytesToHash(c.data.DynastySeed)
}

// DynastySeedLastReset is the last epoch the crosslink seed was reset.
func (c *CrystallizedState) DynastySeedLastReset() uint64 {
	return c.data.DynastySeedLastReset
}

// Validators returns list of validators.
func (c *CrystallizedState) Validators() []*pb.ValidatorRecord {
	return c.data.Validators
}

// ValidatorsLength returns the number of total validators.
func (c *CrystallizedState) ValidatorsLength() int {
	return len(c.data.Validators)
}

// UpdateValidators updates the validator set.
func (c *CrystallizedState) UpdateValidators(validators []*pb.ValidatorRecord) {
	c.data.Validators = validators
}

// IndicesForHeights returns what active validators are part of the attester set
// at what height, and in what shard.
func (c *CrystallizedState) IndicesForHeights() []*pb.ArrayShardAndIndices {
	return c.data.IndicesForHeights
}

// ClearIndicesForHeights clears the IndicesForHeights set.
func (c *CrystallizedState) ClearIndicesForHeights() {
	c.data.IndicesForHeights = []*pb.ArrayShardAndIndices{}
}

// UpdateJustifiedEpoch updates the justified epoch during an epoch transition.
func (c *CrystallizedState) UpdateJustifiedEpoch() {
	epoch := c.LastJustifiedEpoch()
	c.SetLastJustifiedEpoch(c.EpochNumber())

	if c.EpochNumber() == (epoch + 1) {
		c.SetLastFinalizedEpoch(epoch)
	}
}
