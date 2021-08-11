package types

import (
	"github.com/prysmaticlabs/prysm/shared/version"
)

// FieldIndex represents the relevant field position in the
// state struct for a field.
type FieldIndex int

// DataType signifies the data type of the field.
type DataType int

// List of current data types the state supports.
const (
	// BasicArray represents a simple array type for a field.
	BasicArray DataType = iota
	// CompositeArray represents a variable length array with
	// a non primitive type.
	CompositeArray
)

// String returns the name of the field index.
func (f FieldIndex) String(stateVersion int) string {
	switch f {
	case GenesisTime:
		return "genesisTime"
	case GenesisValidatorRoot:
		return "genesisValidatorRoot"
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
		if version.Altair == stateVersion {
			return "previousEpochParticipationBits"
		}
		return "previousEpochAttestations"
	case CurrentEpochAttestations:
		if version.Altair == stateVersion {
			return "currentEpochParticipationBits"
		}
		return "currentEpochAttestations"
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
	default:
		return ""
	}
}

// Below we define a set of useful enum values for the field
// indices of the beacon state. For example, genesisTime is the
// 0th field of the beacon state. This is helpful when we are
// updating the Merkle branches up the trie representation
// of the beacon state. The below field indexes correspond
// to the v1 state.
const (
	GenesisTime FieldIndex = iota
	GenesisValidatorRoot
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
	JustificationBits
	PreviousJustifiedCheckpoint
	CurrentJustifiedCheckpoint
	FinalizedCheckpoint
	// State Fields Added in Altair.
	InactivityScores
	CurrentSyncCommittee
	NextSyncCommittee
)

// State Fields Added in Altair.
const (
	// Epoch Attestations is switched with participation bits in
	// Altair.
	PreviousEpochParticipationBits = PreviousEpochAttestations
	CurrentEpochParticipationBits  = CurrentEpochAttestations
)
