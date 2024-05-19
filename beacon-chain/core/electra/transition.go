package electra

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	e "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch"
)

// Re-exports for methods that haven't changed in Electra.
var (
	InitializePrecomputeValidators       = altair.InitializePrecomputeValidators
	ProcessEpochParticipation            = altair.ProcessEpochParticipation
	ProcessInactivityScores              = altair.ProcessInactivityScores
	ProcessRewardsAndPenaltiesPrecompute = altair.ProcessRewardsAndPenaltiesPrecompute
	ProcessSlashings                     = e.ProcessSlashings
	ProcessEth1DataReset                 = e.ProcessEth1DataReset
	ProcessSlashingsReset                = e.ProcessSlashingsReset
	ProcessRandaoMixesReset              = e.ProcessRandaoMixesReset
	ProcessHistoricalDataUpdate          = e.ProcessHistoricalDataUpdate
	ProcessParticipationFlagUpdates      = altair.ProcessParticipationFlagUpdates
	ProcessSyncCommitteeUpdates          = altair.ProcessSyncCommitteeUpdates
	AttestationsDelta                    = altair.AttestationsDelta
	ProcessSyncAggregate                 = altair.ProcessSyncAggregate
)
