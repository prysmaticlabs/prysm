package slotutil

import (
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "slotutil")

// CountdownToGenesis starts a ticker at the specified duration
// logging the remaining minutes until the genesis chainstart event
// along with important genesis state metadata such as number
// of genesis validators.
func CountdownToGenesis(genesisTime time.Time, genesisValidatorCount uint64) {
	ticker := time.NewTicker(params.BeaconConfig().GenesisCountdownInterval)
	for {
		select {
		case <-time.NewTimer(genesisTime.Sub(roughtime.Now()) + 1).C:
			return
		case <-ticker.C:
			minutesRemaining := genesisTime.Sub(roughtime.Now()).Round(
				time.Minute,
			)/time.Minute + 1
			log.WithFields(logrus.Fields{
				"genesisValidators": fmt.Sprintf("%d", genesisValidatorCount),
				"genesisTime":       fmt.Sprintf("%v", genesisTime),
			}).Infof(
				"%d minute(s) until chain genesis",
				minutesRemaining,
			)
		}
	}

}
