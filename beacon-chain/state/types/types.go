package types

import "reflect"

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
func (f FieldIndex) String() string {
	return reflect.TypeOf(f).Name()
	/*
		switch f {
		case genesisTime:
			return "genesisTime"
		case genesisValidatorRoot:
			return "genesisValidatorRoot"
		case slot:
			return "slot"
		case fork:
			return "fork"
		case latestBlockHeader:
			return "latestBlockHeader"
		case blockRoots:
			return "blockRoots"
		case stateRoots:
			return "stateRoots"
		case historicalRoots:
			return "historicalRoots"
		case eth1Data:
			return "eth1Data"
		case eth1DataVotes:
			return "eth1DataVotes"
		case eth1DepositIndex:
			return "eth1DepositIndex"
		case validators:
			return "validators"
		case balances:
			return "balances"
		case randaoMixes:
			return "randaoMixes"
		case slashings:
			return "slashings"
		case previousEpochAttestations:
			return "previousEpochAttestations"
		case currentEpochAttestations:
			return "currentEpochAttestations"
		case justificationBits:
			return "justificationBits"
		case previousJustifiedCheckpoint:
			return "previousJustifiedCheckpoint"
		case currentJustifiedCheckpoint:
			return "currentJustifiedCheckpoint"
		case finalizedCheckpoint:
			return "finalizedCheckpoint"
		default:
			return ""
		}
	*/
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
)

// State Fields Added in Altair.
const (
	// Epoch Attestations is switched with participation bits in
	// Altair.
	PreviousEpochParticipationBits = PreviousEpochAttestations
	CurrentEpochParticipationBits  = CurrentEpochAttestations
	// Below 3 fields follow on from the finalized checkpoint.
	InactivityScores = iota + FinalizedCheckpoint + 1
	CurrentSyncCommittee
	NextSyncCommittee
)
