package types

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
type ActiveState struct {
	TotalAttesterDeposits uint64 // TotalAttesterDeposits is the total quantity of wei that attested for the most recent checkpoint.
	AttesterBitfields     []byte // AttesterBitfields represents which validator has attested.
}

// PartialCrosslinkRecord contains information about cross links
// that are being put together during this epoch.
type PartialCrosslinkRecord struct {
	ShardID        uint16      // ShardID is the shard crosslink being made for.
	ShardBlockHash common.Hash // ShardBlockHash is the hash of the block.
	NotaryBitfield []byte      // NotaryBitfield determines which notary has voted.
	AggregateSig   []uint      // AggregateSig is the aggregated signature of all the notaries who voted.
}

// CrystallizedState contains fields of every epoch state,
// it changes every epoch.
type CrystallizedState struct {
	ActiveValidators       []ValidatorRecord // ActiveValidators is the list of active validators.
	QueuedValidators       []ValidatorRecord // QueuedValidators is the list of joined but not yet inducted validators.
	ExitedValidators       []ValidatorRecord // ExitedValidators is the list of removed validators pending withdrawal.
	CurrentShuffling       []uint16          // CurrentShuffling is hhe permutation of validators used to determine who cross-links what shard in this epoch.
	CurrentEpoch           uint64            // CurrentEpoch is the current epoch.
	LastJustifiedEpoch     uint64            // LastJustifiedEpoch is the last justified epoch.
	LastFinalizedEpoch     uint64            // LastFinalizedEpoch is the last finalized epoch.
	Dynasty                uint64            // Dynasty is the current dynasty.
	NextShard              uint16            // NextShard is the next shard that cross-linking assignment will start from.
	CurrentCheckpoint      common.Hash       // CurrentCheckpoint is the current FFG checkpoint.
	CrosslinkRecords       []CrosslinkRecord // CrosslinkRecords records about the most recent crosslink for each shard.
	TotalDeposits          uint              // TotalDeposits is the Total balance of deposits.
	CrosslinkSeed          common.Hash       // CrosslinkSeed is used to select the committees for each shard.
	CrosslinkSeedLastReset uint64            // CrosslinkSeedLastReset is the last epoch the crosslink seed was reset.
}

// ValidatorRecord contains information about a validator
type ValidatorRecord struct {
	PubKey            ecdsa.PublicKey // PubKey is the validator's public key.
	WithdrawalShard   uint16          // WithdrawalShard is the shard balance will be sent to after withdrawal.
	WithdrawalAddress common.Address  // WithdrawalAddress is the address balance will be sent to after withdrawal.
	RandaoCommitment  common.Hash     // RandaoCommitment is validator's current RANDAO beacon commitment.
	Balance           uint64          // Balance is validator's current balance.
	SwitchDynasty     uint64          // SwitchDynasty is the dynasty where the validator can (be inducted | be removed | withdraw their balance).
}

// CrosslinkRecord contains the fields of last fully formed
// crosslink to be submitted into the chain.
type CrosslinkRecord struct {
	Epoch uint64      // Epoch records the epoch the crosslink was submitted in.
	Hash  common.Hash // Hash is the block hash.
}

// NewGenesisStates initializes a beacon chain with starting parameters.
func NewGenesisStates() (*ActiveState, *CrystallizedState) {
	active := &ActiveState{
		TotalAttesterDeposits: 0,
		AttesterBitfields:     []byte{},
	}
	crystallized := &CrystallizedState{
		ActiveValidators:   []ValidatorRecord{},
		QueuedValidators:   []ValidatorRecord{},
		ExitedValidators:   []ValidatorRecord{},
		CurrentShuffling:   []uint16{},
		CurrentEpoch:       0,
		LastJustifiedEpoch: 0,
		LastFinalizedEpoch: 0,
		Dynasty:            0,
		TotalDeposits:      0,
		CrosslinkSeed:      common.BytesToHash([]byte{}),
	}
	return active, crystallized
}
