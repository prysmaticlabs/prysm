package blockchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

var canonicalHeadKey = "latest-canonical-head"
var activeStateLookupKey = "beacon-active-state"
var crystallizedStateLookupKey = "beacon-crystallized-state"

var clock utils.Clock = &utils.RealClock{}

// BeaconChain represents the core PoS blockchain object containing
// both a crystallized and active state.
type BeaconChain struct {
	state *beaconState
	lock  sync.Mutex
	db    ethdb.Database
}

type beaconState struct {
	// ActiveState captures the beacon state at block processing level,
	// it focuses on verifying aggregated signatures and pending attestations.
	ActiveState *types.ActiveState
	// CrystallizedState captures the beacon state at cycle transition level,
	// it focuses on changes to the validator set, processing cross links and
	// setting up FFG checkpoints.
	CrystallizedState *types.CrystallizedState
}

// NewBeaconChain initializes a beacon chain using genesis state parameters if
// none provided.
func NewBeaconChain(db ethdb.Database) (*BeaconChain, error) {
	beaconChain := &BeaconChain{
		db:    db,
		state: &beaconState{},
	}
	hasActive, err := db.Has([]byte(activeStateLookupKey))
	if err != nil {
		return nil, err
	}
	hasCrystallized, err := db.Has([]byte(crystallizedStateLookupKey))
	if err != nil {
		return nil, err
	}
	hasGenesis, err := db.Has([]byte("genesis"))
	if err != nil {
		return nil, err
	}
	if !hasGenesis {
		log.Info("No genesis block found on disk, initializing genesis block")
		genesisBlock, err := types.NewGenesisBlock()
		if err != nil {
			return nil, err
		}
		genesisMarshall, err := proto.Marshal(genesisBlock.Proto())
		if err != nil {
			return nil, err
		}
		if err := beaconChain.db.Put([]byte("genesis"), genesisMarshall); err != nil {
			return nil, err
		}
	}
	if !hasActive && !hasCrystallized {
		log.Info("No chainstate found on disk, initializing beacon from genesis")
		active, crystallized, err := types.NewGenesisStates()
		if err != nil {
			return nil, err
		}
		beaconChain.state.ActiveState = active
		beaconChain.state.CrystallizedState = crystallized

		return beaconChain, nil
	}
	if hasActive {
		enc, err := db.Get([]byte(activeStateLookupKey))
		if err != nil {
			return nil, err
		}
		activeData := &pb.ActiveState{}
		err = proto.Unmarshal(enc, activeData)
		if err != nil {
			return nil, err
		}
		beaconChain.state.ActiveState = types.NewActiveState(activeData, make(map[common.Hash]*types.VoteCache))
	}
	if hasCrystallized {
		enc, err := db.Get([]byte(crystallizedStateLookupKey))
		if err != nil {
			return nil, err
		}
		crystallizedData := &pb.CrystallizedState{}
		err = proto.Unmarshal(enc, crystallizedData)
		if err != nil {
			return nil, err
		}
		beaconChain.state.CrystallizedState = types.NewCrystallizedState(crystallizedData)
	}
	return beaconChain, nil
}

// GenesisBlock returns the canonical, genesis block.
func (b *BeaconChain) GenesisBlock() (*types.Block, error) {
	genesisExists, err := b.db.Has([]byte("genesis"))
	if err != nil {
		return nil, err
	}
	if genesisExists {
		bytes, err := b.db.Get([]byte("genesis"))
		if err != nil {
			return nil, err
		}
		block := &pb.BeaconBlock{}
		if err := proto.Unmarshal(bytes, block); err != nil {
			return nil, err
		}
		return types.NewBlock(block), nil
	}
	return types.NewGenesisBlock()
}

// CanonicalHead fetches the latest head stored in persistent storage.
func (b *BeaconChain) CanonicalHead() (*types.Block, error) {
	headExists, err := b.db.Has([]byte(canonicalHeadKey))
	if err != nil {
		return nil, err
	}
	if headExists {
		bytes, err := b.db.Get([]byte(canonicalHeadKey))
		if err != nil {
			return nil, err
		}
		block := &pb.BeaconBlock{}
		if err := proto.Unmarshal(bytes, block); err != nil {
			return nil, fmt.Errorf("cannot unmarshal proto: %v", err)
		}
		return types.NewBlock(block), nil
	}
	return nil, nil
}

// ActiveState contains the current state of attestations and changes every block.
func (b *BeaconChain) ActiveState() *types.ActiveState {
	return b.state.ActiveState
}

// CrystallizedState contains cycle dependent validator information, changes every cycle.
func (b *BeaconChain) CrystallizedState() *types.CrystallizedState {
	return b.state.CrystallizedState
}

// SetActiveState is a convenience method which sets and persists the active state on the beacon chain.
func (b *BeaconChain) SetActiveState(activeState *types.ActiveState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.ActiveState = activeState
	return b.PersistActiveState()
}

// SetCrystallizedState is a convenience method which sets and persists the crystallized state on the beacon chain.
func (b *BeaconChain) SetCrystallizedState(crystallizedState *types.CrystallizedState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.CrystallizedState = crystallizedState
	return b.PersistCrystallizedState()
}

// PersistActiveState stores proto encoding of the current beacon chain active state into the db.
func (b *BeaconChain) PersistActiveState() error {
	encodedState, err := b.ActiveState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(activeStateLookupKey), encodedState)
}

// PersistCrystallizedState stores proto encoding of the current beacon chain crystallized state into the db.
func (b *BeaconChain) PersistCrystallizedState() error {
	encodedState, err := b.CrystallizedState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(crystallizedStateLookupKey), encodedState)
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new crystallized state and active state transition will occur.
func (b *BeaconChain) IsCycleTransition(slotNumber uint64) bool {
	return slotNumber >= b.CrystallizedState().LastStateRecalc()+params.CycleLength
}

// CanProcessBlock is called to decide if an incoming p2p block can be processed into the chain's block trie,
// it checks time stamp, beacon chain parent block hash. It also checks pow chain reference hash if it's a validator.
func (b *BeaconChain) CanProcessBlock(fetcher types.POWBlockFetcher, block *types.Block, isValidator bool) (bool, error) {
	if isValidator {
		if _, err := fetcher.BlockByHash(context.Background(), block.PowChainRef()); err != nil {
			return false, fmt.Errorf("fetching PoW block corresponding to mainchain reference failed: %v", err)
		}
	}

	canProcess, err := b.verifyBlockTimeStamp(block)
	if err != nil {
		return false, fmt.Errorf("unable to process block: %v", err)
	}
	if !canProcess {
		return false, fmt.Errorf("time stamp verification for beacon block %v failed", block.SlotNumber())
	}
	return canProcess, nil
}

// verifyBlockTimeStamp verifies node's local time is greater than or equal to
// min timestamp as computed by GENESIS_TIME + slot_number * SLOT_DURATION.
func (b *BeaconChain) verifyBlockTimeStamp(block *types.Block) (bool, error) {
	slotDuration := time.Duration(block.SlotNumber()*params.SlotDuration) * time.Second
	genesis, err := b.GenesisBlock()
	if err != nil {
		return false, err
	}
	genesisTime, err := genesis.Timestamp()
	if err != nil {
		return false, err
	}
	if clock.Now().Before(genesisTime.Add(slotDuration)) {
		return false, nil
	}
	return true, nil
}

// verifyBlockActiveHash verifies block's active state hash equal to
// node's computed active state hash.
func (b *BeaconChain) verifyBlockActiveHash(block *types.Block) (bool, error) {
	hash, err := b.ActiveState().Hash()
	if err != nil {
		return false, err
	}
	if block.ActiveStateHash() != hash {
		return false, nil
	}
	return true, nil
}

// verifyBlockCrystallizedHash verifies block's crystallized state hash equal to
// node's computed crystallized state hash.
func (b *BeaconChain) verifyBlockCrystallizedHash(block *types.Block) (bool, error) {
	hash, err := b.CrystallizedState().Hash()
	if err != nil {
		return false, err
	}
	if block.CrystallizedStateHash() != hash {
		return false, nil
	}
	return true, nil
}

// computeNewActiveState for every newly processed beacon block.
func (b *BeaconChain) computeNewActiveState(attestations []*pb.AttestationRecord, activeState *types.ActiveState, blockVoteCache map[common.Hash]*types.VoteCache) (*types.ActiveState, error) {
	// TODO: Insert recent block hash.
	activeState.SetBlockVoteCache(blockVoteCache)
	activeState.NewPendingAttestation(attestations)

	return activeState, nil
}

// saveBlock puts the passed block into the beacon chain db.
func (b *BeaconChain) saveBlock(block *types.Block) error {
	encodedState, err := block.Marshal()
	if err != nil {
		return err
	}
	hash, err := block.Hash()
	if err != nil {
		return err
	}

	return b.db.Put(hash[:], encodedState)
}

// processAttestation processes the attestations for one shard in an incoming block.
func (b *BeaconChain) processAttestation(attestationIndex int, block *types.Block) error {
	// Validate attestation's slot number has is within range of incoming block number.
	slotNumber := int(block.SlotNumber())
	attestation := block.Attestations()[attestationIndex]
	if int(attestation.Slot) > slotNumber {
		return fmt.Errorf("attestation slot number can't be higher than block slot number. Found: %d, Needed lower than: %d",
			attestation.Slot,
			slotNumber)
	}
	if int(attestation.Slot) < slotNumber-params.CycleLength {
		return fmt.Errorf("attestation slot number can't be lower than block slot number by one CycleLength. Found: %v, Needed greater than: %v",

			attestation.Slot,
			slotNumber-params.CycleLength)
	}

	// Get all the block hashes up to cycle length.
	parentHashes := b.getSignedParentHashes(block, attestation)
	attesterIndices, err := b.getAttesterIndices(attestation)
	if err != nil {
		return fmt.Errorf("unable to get validator committee: %v", attesterIndices)
	}

	// Verify attester bitfields matches crystallized state's prev computed bitfield.
	if err := b.validateAttesterBitfields(attestation, attesterIndices); err != nil {
		return err
	}

	// TODO: Generate validators aggregated pub key.

	// Hash parentHashes + shardID + slotNumber + shardBlockHash into a message to use to
	// to verify with aggregated public key and aggregated attestation signature.
	msg := make([]byte, binary.MaxVarintLen64)
	var signedHashesStr []byte
	for _, parentHash := range parentHashes {
		signedHashesStr = append(signedHashesStr, parentHash.Bytes()...)
		signedHashesStr = append(signedHashesStr, byte(' '))
	}
	binary.PutUvarint(msg, attestation.Slot%params.CycleLength)
	msg = append(msg, signedHashesStr...)
	binary.PutUvarint(msg, attestation.ShardId)
	msg = append(msg, attestation.ShardBlockHash...)

	msgHash := blake2b.Sum512(msg)

	log.Debugf("Attestation message for shard: %v, slot %v, block hash %v is: %v",
		attestation.ShardId, attestation.Slot, attestation.ShardBlockHash, msgHash)

	// TODO: Verify msgHash against aggregated pub key and aggregated signature.
	return nil
}

// calculateBlockVoteCache calculates and updates active state's block vote cache.
func (b *BeaconChain) calculateBlockVoteCache(attestationIndex int, block *types.Block) (map[common.Hash]*types.VoteCache, error) {
	attestation := block.Attestations()[attestationIndex]
	newVoteCache := b.ActiveState().GetBlockVoteCache()
	parentHashes := b.getSignedParentHashes(block, attestation)
	attesterIndices, err := b.getAttesterIndices(attestation)
	if err != nil {
		return nil, err
	}

	for _, h := range parentHashes {
		// Skip calculating for this hash if the hash is part of oblique parent hashes.
		var skip bool
		for _, obliqueParentHash := range attestation.ObliqueParentHashes {
			if bytes.Equal(h.Bytes(), obliqueParentHash) {
				skip = true
			}
		}
		if skip {
			continue
		}

		// Initialize vote cache of a given block hash if it doesn't exist already.
		if !b.ActiveState().IsVoteCacheEmpty(*h) {
			newVoteCache[*h] = &types.VoteCache{VoterIndices: []uint32{}, VoteTotalDeposit: 0}
		}

		// Loop through attester indices, if the attester has voted but was not accounted for
		// in the cache, then we add attester's index and balance to the block cache.
		for i, attesterIndex := range attesterIndices {
			var existingAttester bool
			if !utils.CheckBit(attestation.AttesterBitfield, i) {
				continue
			}
			for _, indexInCache := range newVoteCache[*h].VoterIndices {
				if attesterIndex == indexInCache {
					existingAttester = true
				}
			}
			if !existingAttester {
				newVoteCache[*h].VoterIndices = append(newVoteCache[*h].VoterIndices, attesterIndex)
				newVoteCache[*h].VoteTotalDeposit += b.CrystallizedState().Validators()[attesterIndex].Balance
			}
		}
	}
	return newVoteCache, nil
}

// getSignedParentHashes returns all the parent hashes stored in active state up to last ycle length.
func (b *BeaconChain) getSignedParentHashes(block *types.Block, attestation *pb.AttestationRecord) []*common.Hash {
	var signedParentHashes []*common.Hash
	start := block.SlotNumber() - attestation.Slot
	end := block.SlotNumber() - attestation.Slot - uint64(len(attestation.ObliqueParentHashes)) + params.CycleLength
	for _, hashes := range b.ActiveState().RecentBlockHashes()[start:end] {
		signedParentHashes = append(signedParentHashes, &hashes)
	}
	for _, obliqueParentHashes := range attestation.ObliqueParentHashes {
		hashes := common.BytesToHash(obliqueParentHashes)
		signedParentHashes = append(signedParentHashes, &hashes)
	}
	return signedParentHashes
}

// getAttesterIndices returns the attester committee of based from attestation's shard ID and slot number.
func (b *BeaconChain) getAttesterIndices(attestation *pb.AttestationRecord) ([]uint32, error) {
	lastStateRecalc := b.CrystallizedState().LastStateRecalc()
	// TODO: IndicesForHeights will return default value because the spec for dynasty transition is not finalized.
	shardCommitteeArray := b.CrystallizedState().IndicesForSlots()
	shardCommittee := shardCommitteeArray[attestation.Slot-lastStateRecalc].ArrayShardAndCommittee
	for i := 0; i < len(shardCommittee); i++ {
		if attestation.ShardId == shardCommittee[i].ShardId {
			return shardCommittee[i].Committee, nil
		}
	}
	return nil, fmt.Errorf("unable to find attestation based on slot: %v, shardID: %v", attestation.Slot, attestation.ShardId)
}

// validateAttesterBitfields validates the attester bitfields are equal between attestation and crystallized state's calculation.
func (b *BeaconChain) validateAttesterBitfields(attestation *pb.AttestationRecord, attesterIndices []uint32) error {
	// Validate attester bit field has the correct length.
	if utils.BitLength(len(attesterIndices)) != len(attestation.AttesterBitfield) {
		return fmt.Errorf("attestation has incorrect bitfield length. Found %v, expected %v",
			len(attestation.AttesterBitfield), utils.BitLength(len(attesterIndices)))
	}

	// Valid attestation can not have non-zero trailing bits.
	lastBit := len(attesterIndices)
	if lastBit%8 != 0 {
		for i := 0; i < 8-lastBit%8; i++ {
			if utils.CheckBit(attestation.AttesterBitfield, lastBit+i) {
				return errors.New("attestation has non-zero trailing bits")
			}
		}
	}
	return nil
}

// saveCanonical puts the passed block into the beacon chain db
// and also saves a "latest-head" key mapping to the block in the db.
func (b *BeaconChain) saveCanonical(block *types.Block) error {
	if err := b.saveBlock(block); err != nil {
		return err
	}
	enc, err := block.Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(canonicalHeadKey), enc)
}

// initCycle is called when a new cycle has been reached, beacon node
// will re-compute active state and crystallized state during init cycle transition.
func (b *BeaconChain) initCycle(cState *types.CrystallizedState, aState *types.ActiveState) (*types.CrystallizedState, *types.ActiveState) {
	var blockVoteBalance uint64
	justifiedStreak := cState.JustifiedStreak()
	justifiedSlot := cState.LastJustifiedSlot()
	finalizedSlot := cState.LastFinalizedSlot()
	blockVoteCache := aState.GetBlockVoteCache()

	// walk through all the slots from LastStateRecalc - cycleLength to LastStateRecalc - 1.
	for i := uint64(0); i < params.CycleLength; i++ {
		slot := cState.LastStateRecalc() - params.CycleLength + i
		blockHash := aState.RecentBlockHashes()[i]
		if _, ok := blockVoteCache[blockHash]; ok {
			blockVoteBalance = blockVoteCache[blockHash].VoteTotalDeposit
		} else {
			blockVoteBalance = 0
		}
		if 3*blockVoteBalance >= 2*cState.TotalDeposits() {
			if slot > justifiedSlot {
				justifiedSlot = slot
			}
			justifiedStreak++
		} else {
			justifiedStreak = 0
		}

		if justifiedStreak >= params.CycleLength+1 {
			if slot-params.CycleLength > finalizedSlot {
				finalizedSlot = slot - params.CycleLength
			}
		}
	}

	// TODO: Process Crosslink records here.
	newCrossLinkRecords := []*pb.CrosslinkRecord{}

	// Remove attestations older than LastStateRecalc.
	var newPendingAttestations []*pb.AttestationRecord
	for _, attestation := range aState.PendingAttestations() {
		if attestation.Slot > cState.LastStateRecalc() {
			newPendingAttestations = append(newPendingAttestations, attestation)
		}
	}

	// TODO: Full rewards and penalties design is not finalized according to the spec.
	rewardedValidators, _ := casper.CalculateRewards(
		aState.PendingAttestations(),
		cState.Validators(),
		cState.CurrentDynasty(),
		cState.TotalDeposits())

	// Get all active validators and calculate total balance for next cycle.
	var nextCycleBalance uint64
	nextCycleValidators := casper.ActiveValidatorIndices(cState.Validators(), cState.CurrentDynasty())
	for _, index := range nextCycleValidators {
		nextCycleBalance += cState.Validators()[index].Balance
	}

	// Construct new crystallized state for cycle transition.
	newCrystallizedState := types.NewCrystallizedState(&pb.CrystallizedState{
		Validators:             rewardedValidators, // TODO: Stub. Static validator set because dynasty transition is not finalized according to the spec.
		LastStateRecalc:        cState.LastStateRecalc() + params.CycleLength,
		IndicesForSlots:        cState.IndicesForSlots(), // TODO: Stub. This will be addresses by shuffling during dynasty transition.
		LastJustifiedSlot:      justifiedSlot,
		JustifiedStreak:        justifiedStreak,
		LastFinalizedSlot:      finalizedSlot,
		CrosslinkingStartShard: 0, // TODO: Stub. Need to see where this epoch left off.
		CrosslinkRecords:       newCrossLinkRecords,
		DynastySeedLastReset:   cState.DynastySeedLastReset(), // TODO: Stub. Dynasty transition is not finalized according to the spec.
		TotalDeposits:          nextCycleBalance,
	})

	var recentBlockHashes [][]byte
	for _, blockHashes := range aState.RecentBlockHashes() {
		recentBlockHashes = append(recentBlockHashes, blockHashes.Bytes())
	}

	// Construct new active state for cycle transition.
	newActiveState := types.NewActiveState(&pb.ActiveState{
		PendingAttestations: newPendingAttestations,
		RecentBlockHashes:   recentBlockHashes,
	}, aState.GetBlockVoteCache())

	return newCrystallizedState, newActiveState
}
