// Package roughtime is a wrapper for a roughtime clock source.
package roughtime

import (
	"context"
	"math"
	"time"

	rt "github.com/cloudflare/roughtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

var offsetHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "roughtime_offset_nsec",
	Help: "The absolute value delta between roughtime computed clock time and the system clock time.",
	Buckets: []float64{
		float64(50 * time.Millisecond),
		float64(100 * time.Millisecond),
		float64(500 * time.Millisecond),
		float64(1 * time.Second),
		float64(2 * time.Second),
		float64(10 * time.Second),
	},
})

func init() {
	recalibrateRoughtime()
	runutil.RunEvery(context.Background(), RecalibrationInterval, recalibrateRoughtime)
}

func recalibrateRoughtime() {
	results := rt.Do(rt.Ecosystem, rt.DefaultQueryAttempts, rt.DefaultQueryTimeout, nil)
	// Compute the average difference between the system's time and the
	// Roughtime responses from the servers, rejecting responses whose radii
	// are larger than 2 seconds.
	var err error
	offset, err = rt.AvgDeltaWithRadiusThresh(results, time.Now(), 2*time.Second)
	if err != nil {
		log.WithError(err).Error("Failed to calculate roughtime offset")
	}
	offsetHistogram.Observe(math.Abs(float64(offset)))
	if offset > 2*time.Second {
		log.WithField("offset", offset).Warn("Roughtime reports your clock is off by more than 2 seconds")
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
