package slotutil

import (
	"context"
	"fmt"
	"strings"
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
	for {
		select {
		case <-time.After(timeTillGenesis):
			return
		case <-ticker.C:
			currentTime := roughtime.Now()
			if currentTime.After(genesisTime) {
				return
			}
			timeRemaining := genesisTime.Sub(currentTime)
			log.WithFields(logrus.Fields{
				"genesisValidators": fmt.Sprintf("%d", genesisValidatorCount),
				"genesisTime":       fmt.Sprintf("%v", genesisTime),
			}).Infof(
				"%s until chain genesis",
				formatDuration(timeRemaining),
			)
		case <-ctx.Done():
			return
		}
	}
}

// Format duration truncates any decimal representation of a time
// duration, if present.
// Example: 5h3m2.023920390s turns into 5h3m2s.
func formatDuration(duration time.Duration) string {
	durationString := fmt.Sprintf("%s", duration)
	decimalIndex := strings.Index(durationString, ".")
	if decimalIndex != -1 {
		return durationString[:decimalIndex] + "s"
	}
	return durationString
}
