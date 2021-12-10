// +build minimal

package field_params

const (
	BlockRootsLength                = 64            // SLOTS_PER_HISTORICAL_ROOT
	StateRootsLength                = 64            // SLOTS_PER_HISTORICAL_ROOT
	RandaoMixesLength               = 64            // EPOCHS_PER_HISTORICAL_VECTOR
	HistoricalRootsLength           = 16777216      // HISTORICAL_ROOTS_LIMIT
	ValidatorRegistryLimit          = 1099511627776 // VALIDATOR_REGISTRY_LIMIT
	Eth1DataVotesLength             = 32            // SLOTS_PER_ETH1_VOTING_PERIOD
	PreviousEpochAttestationsLength = 1024          // MAX_ATTESTATIONS * SLOTS_PER_EPOCH
	CurrentEpochAttestationsLength  = 1024          // MAX_ATTESTATIONS * SLOTS_PER_EPOCH
	SlashingsLength                 = 64            // EPOCHS_PER_SLASHINGS_VECTOR
	SyncCommitteeLength             = 32            // SYNC_COMMITTEE_SIZE
)
