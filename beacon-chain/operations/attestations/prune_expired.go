package attestations

import (
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	prysmTime "github.com/prysmaticlabs/prysm/time"
)

// pruneAttsPool prunes attestations pool on every slot interval.
func (s *Service) pruneAttsPool() {
	ticker := time.NewTicker(s.cfg.pruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.pruneExpiredAtts()
			s.updateMetrics()
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}

// This prunes expired attestations from the pool.
func (s *Service) pruneExpiredAtts() {
	aggregatedAtts := s.cfg.Pool.AggregatedAttestations()
	for _, att := range aggregatedAtts {
		if s.expired(att.Data.Slot) {
			if err := s.cfg.Pool.DeleteAggregatedAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired aggregated attestation")
			}
			expiredAggregatedAtts.Inc()
		}
	}

	if _, err := s.cfg.Pool.DeleteSeenUnaggregatedAttestations(); err != nil {
		log.WithError(err).Error("Cannot delete seen attestations")
	}
	unAggregatedAtts, err := s.cfg.Pool.UnaggregatedAttestations()
	if err != nil {
		log.WithError(err).Error("Could not get unaggregated attestations")
		return
	}
	for _, att := range unAggregatedAtts {
		if s.expired(att.Data.Slot) {
			if err := s.cfg.Pool.DeleteUnaggregatedAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired unaggregated attestation")
			}
			expiredUnaggregatedAtts.Inc()
		}
	}

	blockAtts := s.cfg.Pool.BlockAttestations()
	for _, att := range blockAtts {
		if s.expired(att.Data.Slot) {
			if err := s.cfg.Pool.DeleteBlockAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired block attestation")
			}
		}
		expiredBlockAtts.Inc()
	}
}

// Return true if the input slot has been expired.
// Expired is defined as one epoch behind than current time.
func (s *Service) expired(slot types.Slot) bool {
	expirationSlot := slot + params.BeaconConfig().SlotsPerEpoch
	expirationTime := s.genesisTime + uint64(expirationSlot.Mul(params.BeaconConfig().SecondsPerSlot))
	currentTime := uint64(prysmTime.Now().Unix())
	return currentTime >= expirationTime
}
