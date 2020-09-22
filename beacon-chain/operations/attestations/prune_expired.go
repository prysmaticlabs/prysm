package attestations

import (
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
)

// pruneAttsPool prunes attestations pool on every slot interval.
func (s *Service) pruneAttsPool() {
	ticker := time.NewTicker(s.pruneInterval)
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
	aggregatedAtts := s.pool.AggregatedAttestations()
	for _, att := range aggregatedAtts {
		if s.expired(att.Data.Slot) {
			if err := s.pool.DeleteAggregatedAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired aggregated attestation")
			}
			expiredAggregatedAtts.Inc()
		}
	}

	if _, err := s.pool.DeleteSeenUnaggregatedAttestations(); err != nil {
		log.WithError(err).Error("Cannot delete seen attestations")
	}
	unAggregatedAtts, err := s.pool.UnaggregatedAttestations()
	if err != nil {
		log.WithError(err).Error("Could not get unaggregated attestations")
		return
	}
	for _, att := range unAggregatedAtts {
		if s.expired(att.Data.Slot) {
			if err := s.pool.DeleteUnaggregatedAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired unaggregated attestation")
			}
			expiredUnaggregatedAtts.Inc()
		}
	}

	blockAtts := s.pool.BlockAttestations()
	for _, att := range blockAtts {
		if s.expired(att.Data.Slot) {
			if err := s.pool.DeleteBlockAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired block attestation")
			}
		}
		expiredBlockAtts.Inc()
	}
}

// Return true if the input slot has been expired.
// Expired is defined as one epoch behind than current time.
func (s *Service) expired(slot uint64) bool {
	expirationSlot := slot + params.BeaconConfig().SlotsPerEpoch
	expirationTime := s.genesisTime + expirationSlot*params.BeaconConfig().SecondsPerSlot
	currentTime := uint64(timeutils.Now().Unix())
	return currentTime >= expirationTime
}
