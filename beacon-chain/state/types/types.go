package types

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
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
	// CompressedArray represents a variable length array which
	// can pack multiple elements into a leaf of the underlying
	// trie.
	CompressedArray
)

// BeaconStateField represents a field of the beacon state.
type BeaconStateField interface {
	String(stateVersion int) string
	RealPosition() int
	ElemsInChunk() (uint64, error)
	Native() bool
}

// String returns the name of the field index.
func (f FieldIndex) String(stateVersion int) string {
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
		if version.Altair == stateVersion || version.Bellatrix == stateVersion {
			return "previousEpochParticipationBits"
		}
		return "previousEpochAttestations"
	case CurrentEpochAttestations:
		if version.Altair == stateVersion || version.Bellatrix == stateVersion {
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
	case LatestExecutionPayloadHeader:
		return "latestExecutionPayloadHeader"
	default:
		return ""
	}
}

// RealPosition denotes the position of the field in the beacon state.
// The value might differ for different state versions.
func (f FieldIndex) RealPosition() int {
	return int(f)
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
	return false
}

// Below we define a set of useful enum values for the field
// indices of the beacon state. For example, genesisTime is the
// 0th field of the beacon state. This is helpful when we are
// updating the Merkle branches up the trie representation
// of the beacon state. The below field indexes correspond
// to the v1 state.
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
	JustificationBits
	PreviousJustifiedCheckpoint
	CurrentJustifiedCheckpoint
	FinalizedCheckpoint
	// State Fields Added in Altair.
	InactivityScores
	CurrentSyncCommittee
	NextSyncCommittee
	// State fields added in Bellatrix.
	LatestExecutionPayloadHeader
)

// Altair fields which replaced previous phase 0 fields.
const (
	// Epoch Attestations is switched with participation bits in Altair.
	PreviousEpochParticipationBits = PreviousEpochAttestations
	CurrentEpochParticipationBits  = CurrentEpochAttestations
)
