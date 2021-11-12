package monitor

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField("prefix", "monitor")
)

// ValidatorMonitorConfig contains the list of validator indices that the
// monitor service tracks, as well as the event feed notifier that the
// monitor needs to subscribe.
type ValidatorMonitorConfig struct {
	TrackedValidators []types.ValidatorIndex
}

// Service is the main structure that tracks validators and reports logs and
// metrics of their performances throughout their lifetime.
type Service struct {
	config *ValidatorMonitorConfig
}

// TrackedIndex returns if the given validator index corresponds to one of the
// validators we follow
func (s *Service) TrackedIndex(idx types.ValidatorIndex) bool {
	for _, tracked := range s.config.TrackedValidators {
		if tracked == idx {
			return true
		}
	}
	return false
}
