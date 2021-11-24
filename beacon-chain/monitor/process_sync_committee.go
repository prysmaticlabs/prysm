package monitor

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/sirupsen/logrus"
)

// processSyncCommitteeContribution logs the event that one of our tracked
// validators' aggregated sync contribution has been processed.
// TODO: We do not log if a sync contribution was included in an aggregate (we
// log them when they are included in blocks)
func (s *Service) processSyncCommitteeContribution(contribution *ethpb.SignedContributionAndProof) {
	idx := contribution.Message.AggregatorIndex
	s.Lock()
	defer s.Unlock()
	if s.trackedIndex(idx) {
		aggPerf := s.aggregatedPerformance[idx]
		aggPerf.totalSyncComitteeAggregations++
		s.aggregatedPerformance[idx] = aggPerf

		log.WithField("ValidatorIndex", contribution.Message.AggregatorIndex).Info("Sync committee aggregation processed")
	}
}

// processSyncAggregate logs the event that one of our tracked validators is a sync-committee member and its
// contribution was included
func (s *Service) processSyncAggregate(state state.BeaconState, blk block.BeaconBlock) {
	if blk == nil || blk.Body() == nil {
		return
	}
	bits, err := blk.Body().SyncAggregate()
	if err != nil {
		log.WithError(err).Error("Cannot get SyncAggregate")
		return
	}
	s.Lock()
	defer s.Unlock()
	for validatorIdx, committeeIndices := range s.trackedSyncCommitteeIndices {
		if len(committeeIndices) > 0 {
			contrib := 0
			for _, idx := range committeeIndices {
				if bits.SyncCommitteeBits.BitAt(uint64(idx)) {
					contrib++
				}
			}

			balance, err := state.BalanceAtIndex(validatorIdx)
			if err != nil {
				log.Error("Could not get balance")
				return
			}

			latestPerf := s.latestPerformance[validatorIdx]
			balanceChg := int64(balance - latestPerf.balance)
			latestPerf.balanceChange = balanceChg
			latestPerf.balance = balance
			s.latestPerformance[validatorIdx] = latestPerf

			aggPerf := s.aggregatedPerformance[validatorIdx]
			aggPerf.totalSyncComitteeContributions += uint64(contrib)
			s.aggregatedPerformance[validatorIdx] = aggPerf

			syncCommitteeContributionCounter.WithLabelValues(
				fmt.Sprintf("%d", validatorIdx)).Add(float64(contrib))

			log.WithFields(logrus.Fields{
				"ValidatorIndex":  validatorIdx,
				"ExpectedContrib": len(committeeIndices),
				"Contributions":   contrib,
				"NewBalance":      balance,
				"BalanceChange":   balanceChg,
			}).Info("Sync committee contribution included")
		}
	}
}
