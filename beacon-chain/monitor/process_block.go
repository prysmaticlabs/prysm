package monitor

import (
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/sirupsen/logrus"
)

// processSlashings logs the event of one of our tracked validators was slashed
func (s *Service) processSlashings(blk block.BeaconBlock) {
	for _, slashing := range blk.Body().ProposerSlashings() {
		idx := slashing.Header_1.Header.ProposerIndex
		if s.TrackedIndex(idx) {
			log.WithFields(logrus.Fields{
				"ProposerIndex":  idx,
				"SlashedInSlot:": blk.Slot(),
				"Slot":           slashing.Header_1.Header.Slot,
				"Root1":          fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Header_1.Header.BodyRoot)),
				"Root2":          fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Header_2.Header.BodyRoot)),
			}).Info("Proposer slashing was included")
		}
	}

	for _, slashing := range blk.Body().AttesterSlashings() {
		for _, idx := range blocks.SlashableAttesterIndices(slashing) {
			if s.TrackedIndex(types.ValidatorIndex(idx)) {
				log.WithFields(logrus.Fields{
					"AttesterIndex": idx,
					"Slot1":         slashing.Attestation_1.Data.Slot,
					"Root1":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Attestation_1.Data.BeaconBlockRoot)),
					"Source1":       slashing.Attestation_1.Data.Source.Epoch,
					"Target1":       slashing.Attestation_1.Data.Target.Epoch,
					"Slot2":         slashing.Attestation_2.Data.Slot,
					"Root2":         fmt.Sprintf("%#x", bytesutil.Trunc(slashing.Attestation_2.Data.BeaconBlockRoot)),
					"Source2":       slashing.Attestation_2.Data.Source.Epoch,
					"Target2":       slashing.Attestation_2.Data.Target.Epoch,
				}).Info("Attester slashing was included")

			}
		}
	}
}
