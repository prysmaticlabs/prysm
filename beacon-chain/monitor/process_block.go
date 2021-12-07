package monitor

import (
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

// Number of epochs between aggregate reports
const AggregateReportingPeriod = 5

// processBlock handles the cases when
// 1) A block was proposed by one of our tracked validators
// 2) An attestation by one of our tracked validators was included
// 3) An Exit by one of our validators was included
// 4) A Slashing by one of our tracked validators was included
// 5) A Sync Committe Contribution by one of our tracked validators was included
func (s *Service) processBlock(ctx context.Context, b block.SignedBeaconBlock) {
	if b == nil || b.Block() == nil {
		return
	}
	blk := b.Block()

	s.processSlashings(blk)
	s.processExitsFromBlock(blk)

	root, err := blk.HashTreeRoot()
	if err != nil {
		log.WithError(err).Error("Could not compute block's hash tree root")
		return
	}
	state := s.config.StateGen.StateByRootIfCachedNoCopy(root)
	if state == nil {
		log.WithField("BeaconBlockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debug(
			"Skipping block collection due to state not found in cache")
		return
	}

	currEpoch := slots.ToEpoch(blk.Slot())
	s.RLock()
	lastSyncedEpoch := s.lastSyncedEpoch
	s.RUnlock()

	if currEpoch != lastSyncedEpoch &&
		slots.SyncCommitteePeriod(currEpoch) == slots.SyncCommitteePeriod(lastSyncedEpoch) {
		s.updateSyncCommitteeTrackedVals(state)
	}

	s.processSyncAggregate(state, blk)
	s.processProposedBlock(state, root, blk)
	s.processAttestations(ctx, state, blk)

	if blk.Slot()%(AggregateReportingPeriod*params.BeaconConfig().SlotsPerEpoch) == 0 {
		s.logAggregatedPerformance()
	}
}

// processProposedBlock logs the event that one of our tracked validators proposed a block that was included
func (s *Service) processProposedBlock(state state.BeaconState, root [32]byte, blk block.BeaconBlock) {
	s.Lock()
	defer s.Unlock()
	if s.trackedIndex(blk.ProposerIndex()) {
		// update metrics
		proposedSlotsCounter.WithLabelValues(fmt.Sprintf("%d", blk.ProposerIndex())).Inc()

		// update the performance map
		balance, err := state.BalanceAtIndex(blk.ProposerIndex())
		if err != nil {
			log.WithError(err).Error("Could not get balance")
			return
		}

		latestPerf := s.latestPerformance[blk.ProposerIndex()]
		balanceChg := int64(balance - latestPerf.balance)
		latestPerf.balanceChange = balanceChg
		latestPerf.balance = balance
		s.latestPerformance[blk.ProposerIndex()] = latestPerf

		aggPerf := s.aggregatedPerformance[blk.ProposerIndex()]
		aggPerf.totalProposedCount++
		s.aggregatedPerformance[blk.ProposerIndex()] = aggPerf

		log.WithFields(logrus.Fields{
			"ProposerIndex": blk.ProposerIndex(),
			"Slot":          blk.Slot(),
			"Version":       blk.Version(),
			"ParentRoot":    fmt.Sprintf("%#x", bytesutil.Trunc(blk.ParentRoot())),
			"BlockRoot":     fmt.Sprintf("%#x", bytesutil.Trunc(root[:])),
			"NewBalance":    balance,
			"BalanceChange": balanceChg,
		}).Info("Proposed block was included")
	}
}

// processSlashings logs the event of one of our tracked validators was slashed
func (s *Service) processSlashings(blk block.BeaconBlock) {
	s.RLock()
	defer s.RUnlock()
	for _, slashing := range blk.Body().ProposerSlashings() {
		idx := slashing.Header_1.Header.ProposerIndex
		if s.trackedIndex(idx) {
			log.WithFields(logrus.Fields{
				"ProposerIndex": idx,
				"Slot":          blk.Slot(),
				"SlashingSlot":  slashing.Header_1.Header.Slot,
				"Root1":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Header_1.Header.BodyRoot)),
				"Root2":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Header_2.Header.BodyRoot)),
			}).Info("Proposer slashing was included")
		}
	}

	for _, slashing := range blk.Body().AttesterSlashings() {
		for _, idx := range blocks.SlashableAttesterIndices(slashing) {
			if s.trackedIndex(types.ValidatorIndex(idx)) {
				log.WithFields(logrus.Fields{
					"AttesterIndex": idx,
					"Slot":          blk.Slot(),
					"Slot1":         slashing.Attestation_1.Data.Slot,
					"Root1":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Attestation_1.Data.BeaconBlockRoot)),
					"SourceEpoch1":  slashing.Attestation_1.Data.Source.Epoch,
					"TargetEpoch1":  slashing.Attestation_1.Data.Target.Epoch,
					"Slot2":         slashing.Attestation_2.Data.Slot,
					"Root2":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Attestation_2.Data.BeaconBlockRoot)),
					"SourceEpoch2":  slashing.Attestation_2.Data.Source.Epoch,
					"TargetEpoch2":  slashing.Attestation_2.Data.Target.Epoch,
				}).Info("Attester slashing was included")

			}
		}
	}
}

// logAggregatedPerformance logs the performance statistics collected since the run started
func (s *Service) logAggregatedPerformance() {
	s.RLock()
	defer s.RUnlock()

	for idx, p := range s.aggregatedPerformance {
		if p.totalAttestedCount == 0 || p.totalRequestedCount == 0 || p.startBalance == 0 {
			break
		}
		l, ok := s.latestPerformance[idx]
		if !ok {
			break
		}
		percentAtt := float64(p.totalAttestedCount) / float64(p.totalRequestedCount)
		percentBal := float64(l.balance-p.startBalance) / float64(p.startBalance)
		percentDistance := float64(p.totalDistance) / float64(p.totalAttestedCount)
		percentCorrectSource := float64(p.totalCorrectSource) / float64(p.totalAttestedCount)
		percentCorrectHead := float64(p.totalCorrectHead) / float64(p.totalAttestedCount)
		percentCorrectTarget := float64(p.totalCorrectTarget) / float64(p.totalAttestedCount)

		log.WithFields(logrus.Fields{
			"ValidatorIndex":           idx,
			"StartEpoch":               p.startEpoch,
			"StartBalance":             p.startBalance,
			"TotalRequested":           p.totalRequestedCount,
			"AttestationInclusion":     fmt.Sprintf("%.2f%%", percentAtt*100),
			"BalanceChangePct":         fmt.Sprintf("%.2f%%", percentBal*100),
			"CorrectlyVotedSourcePct":  fmt.Sprintf("%.2f%%", percentCorrectSource*100),
			"CorrectlyVotedTargetPct":  fmt.Sprintf("%.2f%%", percentCorrectTarget*100),
			"CorrectlyVotedHeadPct":    fmt.Sprintf("%.2f%%", percentCorrectHead*100),
			"AverageInclusionDistance": fmt.Sprintf("%.1f", percentDistance),
			"TotalProposedBlocks":      p.totalProposedCount,
			"TotalAggregations":        p.totalAggregations,
			"TotalSyncContributions":   p.totalSyncComitteeContributions,
		}).Info("Aggregated performance since launch")
	}
}
