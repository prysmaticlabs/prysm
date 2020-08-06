// Package roughtime is a wrapper for a roughtime clock source.
package roughtime

import (
	"math"
	"time"

	rt "github.com/cloudflare/roughtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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

var offsetsRejected = promauto.NewCounter(prometheus.CounterOpts{
	Name: "roughtime_offsets_rejected",
	Help: "The number of times that roughtime results could not be verified and the returned offset was rejected",
})

func init() {
	go func() {
		time.Sleep(1 * time.Second)
		if featureconfig.Get().EnableRoughtime {
			recalibrateRoughtime()
			for {
				wait := RecalibrationInterval
				// recalibrate every minute if there is a large skew.
				if offset > 2*time.Second {
					wait = 1 * time.Minute
				}
				select {
				case <-time.After(wait):
					recalibrateRoughtime()
				}
			}
		}
	}()
}

func recalibrateRoughtime() {
	t0 := time.Now()
	results := rt.Do(rt.Ecosystem, rt.DefaultQueryAttempts, rt.DefaultQueryTimeout, nil)

	// Log Debug Results.
	for _, res := range results {
		if res.Error() != nil {
			log.Errorf("Could not get rough time result: %v", res.Error())
			continue
		}
		log.WithFields(logrus.Fields{
			"Server Name": res.Server.Name,
			"Midpoint":    res.Midpoint,
			"Delay":       res.Delay,
			"Radius":      res.Roughtime.Radius,
		}).Debug("Response received from roughtime server")
	}
	// Compute the average difference between the system's time and the
	// Roughtime responses from the servers, rejecting responses whose radii
	// are larger than 2 seconds.
	newOffset, err := rt.AvgDeltaWithRadiusThresh(results, t0, 2*time.Second)
	if err != nil {
		log.WithError(err).Error("Failed to calculate roughtime offset")
	}
	offsetHistogram.Observe(math.Abs(float64(newOffset)))
	if newOffset > 2*time.Second {
		log.WithField("offset", newOffset).Warn("Roughtime reports your clock is off by more than 2 seconds")
	}

	chain := rt.NewChain(results)
	ok, err := chain.Verify(nil)
	if err != nil || !ok {
		log.WithError(err).WithField("offset", newOffset).Error("Could not verify roughtime responses, not accepting roughtime offset")
		offsetsRejected.Inc()
		return
	}

	log.Debugf("New calculated roughtime offset is %d ns", newOffset.Nanoseconds())
	offset = newOffset
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
