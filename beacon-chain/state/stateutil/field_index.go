package stateutil

// FieldIndex to be used for the beacon state trie.
type FieldIndex int

// Below we define a set of useful enum values for the field
// indices of the beacon state. For example, genesisTime is the
// 0th field of the beacon state. This is helpful when we are
// updating the Merkle branches up the trie representation
// of the beacon state.
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

// String returns the name of the field index.
func (f FieldIndex) String() string {
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
		return "previousEpochAttestations"
	case CurrentEpochAttestations:
		return "currentEpochAttestations"
	case JustificationBits:
		return "justificationBits"
	case PreviousJustifiedCheckpoint:
		return "previousJustifiedCheckpoint"
	case CurrentJustifiedCheckpoint:
		return "currentJustifiedCheckpoint"
	case FinalizedCheckpoint:
		return "finalizedCheckpoint"
	default:
		return ""
	}
}
