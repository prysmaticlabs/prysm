// Package roughtime is a wrapper for a roughtime clock source.
package roughtime

import (
	"context"
	"time"

	rt "github.com/cloudflare/roughtime"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/sirupsen/logrus"
)

// RecalibrationInterval for roughtime and system time differences. Set
// as a default of once per hour.
const RecalibrationInterval = time.Hour

// offset is the difference between the system time and the time returned by
// the roughtime server
var offset time.Duration

var log = logrus.WithField("prefix", "roughtime")

func init() {
	runutil.RunEvery(context.Background(), RecalibrationInterval, recalibrateRoughtime)
}

func recalibrateRoughtime() {
	t0 := time.Now()
	results := rt.Do(rt.Ecosystem, rt.DefaultQueryAttempts, rt.DefaultQueryTimeout, nil)
	// Compute the average difference between the system's time and the
	// Roughtime responses from the servers, rejecting responses whose radii
	// are larger than 2 seconds.
	var err error
	offset, err = rt.AvgDeltaWithRadiusThresh(results, t0, 2*time.Second)
	if err != nil {
		log.WithError(err).Error("Failed to calculate roughtime offset")
	}
}

// Since returns the duration since t, based on the roughtime response
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}

// Until returns the duration until t, based on the roughtime response
func Until(t time.Time) time.Duration {
	return t.Sub(Now())
}

// Now returns the current local time given the roughtime offset.
func Now() time.Time {
	return time.Now().Add(offset)
}
