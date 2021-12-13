// +build !minimal

package field_params

const (
	BlockRootsLength                = 8192          // SLOTS_PER_HISTORICAL_ROOT
	StateRootsLength                = 8192          // SLOTS_PER_HISTORICAL_ROOT
	RandaoMixesLength               = 65536         // EPOCHS_PER_HISTORICAL_VECTOR
	HistoricalRootsLength           = 16777216      // HISTORICAL_ROOTS_LIMIT
	ValidatorRegistryLimit          = 1099511627776 // VALIDATOR_REGISTRY_LIMIT
	Eth1DataVotesLength             = 2048          // SLOTS_PER_ETH1_VOTING_PERIOD
	PreviousEpochAttestationsLength = 4096          // MAX_ATTESTATIONS * SLOTS_PER_EPOCH
	CurrentEpochAttestationsLength  = 4096          // MAX_ATTESTATIONS * SLOTS_PER_EPOCH
	SlashingsLength                 = 8192          // EPOCHS_PER_SLASHINGS_VECTOR
	SyncCommitteeLength             = 512           // SYNC_COMMITTEE_SIZE
)
