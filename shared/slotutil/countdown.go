package slotutil

import (
	"time"

	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "slotutil")

// CountdownToGenesis starts a ticker at the specified duration
// logging the remaining minutes until the genesis chainstart event
// along with important genesis state metadata such as number
// of genesis validators.
func CountdownToGenesis(genesisTime time.Time, duration time.Duration) {
	ticker := time.NewTicker(duration * time.Second)
	for {
		select {
		case <-time.NewTimer(genesisTime.Sub(roughtime.Now()) + 1).C:
			return
		case <-ticker.C:
			log.Infof("%02d minutes to genesis!", genesisTime.Sub(roughtime.Now()).Round(
				time.Minute,
			)/time.Minute+1)
		}
	}

}
