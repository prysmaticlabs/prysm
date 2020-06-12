package slotutil

import (
	"context"
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
func CountdownToGenesis(ctx context.Context, genesisTime time.Time, genesisValidatorCount uint64) {
	ticker := time.NewTicker(params.BeaconConfig().GenesisCountdownInterval)
	timeTillGenesis := genesisTime.Sub(roughtime.Now())
	logFields := logrus.Fields{
		"genesisValidators": fmt.Sprintf("%d", genesisValidatorCount),
		"genesisTime":       fmt.Sprintf("%v", genesisTime),
	}
	for {
		select {
		case <-time.After(timeTillGenesis):
			log.WithFields(logFields).Info("Chain genesis time reached")
			return
		case <-ticker.C:
			currentTime := roughtime.Now()
			if currentTime.After(genesisTime) {
				log.WithFields(logFields).Info("Chain genesis time reached")
				return
			}
			timeRemaining := genesisTime.Sub(currentTime)
			if timeRemaining <= 2*time.Minute {
				ticker = time.NewTicker(time.Second)
			}
			if timeRemaining >= time.Second {
				log.WithFields(logFields).Infof(
					"%s until chain genesis",
					timeRemaining.Truncate(time.Second),
				)
			}
		case <-ctx.Done():
			return
		}
	}
}
