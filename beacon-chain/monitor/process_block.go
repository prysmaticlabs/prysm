package monitor

import (
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

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
	s.monitorLock.RLock()
	lastSyncedEpoch := s.lastSyncedEpoch
	s.monitorLock.RUnlock()

	if currEpoch != lastSyncedEpoch &&
		slots.SyncCommitteePeriod(currEpoch) == slots.SyncCommitteePeriod(s.lastSyncedEpoch) {
		s.updateSyncCommitteeTrackedVals(state)
	}

	s.processSyncAggregate(state, blk)
	s.processProposedBlock(state, root, blk)
	s.processAttestations(ctx, state, blk)
}

// processProposedBlock logs the event that one of our tracked validators proposed a block that was included
func (s *Service) processProposedBlock(state state.BeaconState, root [32]byte, blk block.BeaconBlock) {
	if s.TrackedIndex(blk.ProposerIndex()) {
		// update metrics
		proposedSlotsCounter.WithLabelValues(fmt.Sprintf("%d", blk.ProposerIndex())).Inc()

		// update the performance map
		balance, err := state.BalanceAtIndex(blk.ProposerIndex())
		if err != nil {
			log.WithError(err).Error("Could not get balance")
			return
		}

		s.monitorLock.Lock()
		defer s.monitorLock.Unlock()
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
	for _, slashing := range blk.Body().ProposerSlashings() {
		idx := slashing.Header_1.Header.ProposerIndex
		if s.TrackedIndex(idx) {
			log.WithFields(logrus.Fields{
				"ProposerIndex": idx,
				"Slot:":         blk.Slot(),
				"SlashingSlot":  slashing.Header_1.Header.Slot,
				"Root1":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Header_1.Header.BodyRoot)),
				"Root2":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Header_2.Header.BodyRoot)),
			}).Info("Proposer slashing was included")
		}
	}

	for _, slashing := range blk.Body().AttesterSlashings() {
		for _, idx := range blocks.SlashableAttesterIndices(slashing) {
			if s.TrackedIndex(types.ValidatorIndex(idx)) {
				log.WithFields(logrus.Fields{
					"AttesterIndex": idx,
					"Slot:":         blk.Slot(),
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
