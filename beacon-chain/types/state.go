package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
type ActiveState struct {
	TotalAttesterDeposits uint64 // TotalAttesterDeposits is the total quantity of wei that attested for the most recent checkpoint.
	AttesterBitfields     []byte // AttesterBitfields represents which validator has attested.
}

// CrystallizedState contains fields of every epoch state,
// it changes every epoch.
type CrystallizedState struct {
	ActiveValidators   []ValidatorRecord // ActiveValidators is the list of active validators.
	QueuedValidators   []ValidatorRecord // QueuedValidators is the list of joined but not yet inducted validators.
	ExitedValidators   []ValidatorRecord // ExitedValidators is the list of removed validators pending withdrawal.
	CurrentShuffling   []uint16          // CurrentShuffling is hhe permutation of validators used to determine who cross-links what shard in this epoch.
	CurrentEpoch       uint64            // CurrentEpoch is the current epoch.
	LastJustifiedEpoch uint64            // LastJustifiedEpoch is the last justified epoch.
	LastFinalizedEpoch uint64            // LastFinalizedEpoch is the last finalized epoch.
	Dynasty            uint64            // Dynasty is the current dynasty.
	NextShard          uint16            // NextShard is the next shard that cross-linking assignment will start from.
	CurrentCheckpoint  common.Hash       // CurrentCheckpoint is the current FFG checkpoint.
	TotalDeposits      uint              // TotalDeposits is the Total balance of deposits.
}

// ValidatorRecord contains information about a validator
type ValidatorRecord struct {
	PubKey            enr.Secp256k1  // PubKey is the validator's public key.
	WithdrawalShard   uint16         // WithdrawalShard is the shard balance will be sent to after withdrawal.
	WithdrawalAddress common.Address // WithdrawalAddress is the address balance will be sent to after withdrawal.
	RandaoCommitment  common.Hash    // RandaoCommitment is validator's current RANDAO beacon commitment.
	Balance           uint64         // Balance is validator's current balance.
	SwitchDynasty     uint64         // SwitchDynasty is the dynasty where the validator can (be inducted | be removed | withdraw their balance).
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
	}
	return active, crystallized
}
