package attestations

import (
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// Clean attestations pool at certain interval.
var cleanAttsPoolPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second

// This cleans attestations pool by running cleanAtts
// every cleanAttsPoolPeriod.
func (s *Service) cleanAttsPool() {
	ticker := time.NewTicker(cleanAttsPoolPeriod)
	for {
		select {
		case <-ticker.C:
			s.cleanAtts()
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}

func (s *Service) cleanAtts() {
	aggregatedAtts := s.pool.AggregatedAttestations()
	unAggregatedAtts := s.pool.UnaggregatedAttestations()
	blockAtts := s.pool.BlockAttestations()

	for _, att := range aggregatedAtts {
		if s.expired(att.Data.Slot) {
			if err := s.pool.DeleteAggregatedAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired aggregated attestation")
			}
			expiredAggregatedAtts.Inc()
		}
	}

	for _, att := range unAggregatedAtts {
		if s.expired(att.Data.Slot) {
			if err := s.pool.DeleteUnaggregatedAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired unaggregated attestation")
			}
			expiredUnaggregatedAtts.Inc()
		}
	}

	for _, att := range blockAtts {
		if s.expired(att.Data.Slot) {
			if err := s.pool.DeleteBlockAttestation(att); err != nil {
				log.WithError(err).Error("Could not delete expired block attestation")
			}
		}
		expiredBlockAtts.Inc()
	}
}

func (s *Service) expired(slot uint64) bool {
	expirationSlot := slot + params.BeaconConfig().SlotsPerEpoch
	slotExpirationTime := s.genesisTime + expirationSlot*params.BeaconConfig().SecondsPerSlot
	currentTime := uint64(roughtime.Now().Unix())
	if currentTime >= slotExpirationTime {
		return true
	}
	return false
}
