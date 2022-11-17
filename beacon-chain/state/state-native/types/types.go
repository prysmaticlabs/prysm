package types

import (
	"github.com/pkg/errors"
)

// FieldIndex represents the relevant field position in the
// state struct for a field.
type FieldIndex int

// String returns the name of the field index.
func (f FieldIndex) String(_ int) string {
	switch f {
	case GenesisTime:
		return "genesisTime"
	case GenesisValidatorsRoot:
		return "genesisValidatorsRoot"
	case Slot:
		return "slot"
	case Fork:
		return "fork"
	case LatestBlockHeader:
		return "latestBlockHeader"
	case BlockRoots:
		return "blockRoots"
	case StateRoots:
		return "stateRoots"
	case HistoricalRoots:
		return "historicalRoots"
	case Eth1Data:
		return "eth1Data"
	case Eth1DataVotes:
		return "eth1DataVotes"
	case Eth1DepositIndex:
		return "eth1DepositIndex"
	case Validators:
		return "validators"
	case Balances:
		return "balances"
	case RandaoMixes:
		return "randaoMixes"
	case Slashings:
		return "slashings"
	case PreviousEpochAttestations:
		return "previousEpochAttestations"
	case CurrentEpochAttestations:
		return "currentEpochAttestations"
	case PreviousEpochParticipationBits:
		return "previousEpochParticipationBits"
	case CurrentEpochParticipationBits:
		return "currentEpochParticipationBits"
	case JustificationBits:
		return "justificationBits"
	case PreviousJustifiedCheckpoint:
		return "previousJustifiedCheckpoint"
	case CurrentJustifiedCheckpoint:
		return "currentJustifiedCheckpoint"
	case FinalizedCheckpoint:
		return "finalizedCheckpoint"
	case InactivityScores:
		return "inactivityScores"
	case CurrentSyncCommittee:
		return "currentSyncCommittee"
	case NextSyncCommittee:
		return "nextSyncCommittee"
	case LatestExecutionPayloadHeader:
		return "latestExecutionPayloadHeader"
	case LatestExecutionPayloadHeaderCapella:
		return "LatestExecutionPayloadHeaderCapella"
	case NextWithdrawalIndex:
		return "NextWithdrawalIndex"
	case NextWithdrawalValidatorIndex:
		return "LastWithdrawalValidatorIndex"
	default:
		return ""
	}
}

// RealPosition denotes the position of the field in the beacon state.
// The value might differ for different state versions.
func (f FieldIndex) RealPosition() int {
	switch f {
	case GenesisTime:
		return 0
	case GenesisValidatorsRoot:
		return 1
	case Slot:
		return 2
	case Fork:
		return 3
	case LatestBlockHeader:
		return 4
	case BlockRoots:
		return 5
	case StateRoots:
		return 6
	case HistoricalRoots:
		return 7
	case Eth1Data:
		return 8
	case Eth1DataVotes:
		return 9
	case Eth1DepositIndex:
		return 10
	case Validators:
		return 11
	case Balances:
		return 12
	case RandaoMixes:
		return 13
	case Slashings:
		return 14
	case PreviousEpochAttestations, PreviousEpochParticipationBits:
		return 15
	case CurrentEpochAttestations, CurrentEpochParticipationBits:
		return 16
	case JustificationBits:
		return 17
	case PreviousJustifiedCheckpoint:
		return 18
	case CurrentJustifiedCheckpoint:
		return 19
	case FinalizedCheckpoint:
		return 20
	case InactivityScores:
		return 21
	case CurrentSyncCommittee:
		return 22
	case NextSyncCommittee:
		return 23
	case LatestExecutionPayloadHeader, LatestExecutionPayloadHeaderCapella:
		return 24
	case NextWithdrawalIndex:
		return 25
	case NextWithdrawalValidatorIndex:
		return 26
	default:
		return -1
	}
}

// ElemsInChunk returns the number of elements in the chunk (number of
// elements that are able to be packed).
func (f FieldIndex) ElemsInChunk() (uint64, error) {
	switch f {
	case Balances:
		return 4, nil
	default:
		return 0, errors.Errorf("field %d doesn't support element compression", f)
	}
}

func (FieldIndex) Native() bool {
	return true
}

// Below we define a set of useful enum values for the field
// indices of the beacon state. For example, genesisTime is the
// 0th field of the beacon state. This is helpful when we are
// updating the Merkle branches up the trie representation
// of the beacon state. The below field indexes correspond
// to the state.
const (
	GenesisTime FieldIndex = iota
	GenesisValidatorsRoot
	Slot
	Fork
	LatestBlockHeader
	BlockRoots
	StateRoots
	HistoricalRoots
	Eth1Data
	Eth1DataVotes
	Eth1DepositIndex
	Validators
	Balances
	RandaoMixes
	Slashings
	PreviousEpochAttestations
	CurrentEpochAttestations
	PreviousEpochParticipationBits
	CurrentEpochParticipationBits
	JustificationBits
	PreviousJustifiedCheckpoint
	CurrentJustifiedCheckpoint
	FinalizedCheckpoint
	InactivityScores
	CurrentSyncCommittee
	NextSyncCommittee
	LatestExecutionPayloadHeader
	LatestExecutionPayloadHeaderCapella
	NextWithdrawalIndex
	NextWithdrawalValidatorIndex
)
