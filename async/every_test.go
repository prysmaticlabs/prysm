package async_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/async"
	"github.com/prysmaticlabs/prysm/v4/config/params"
)

func TestEveryRuns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	i := int32(0)
	async.RunEvery(ctx, 100*time.Millisecond, func() {
		atomic.AddInt32(&i, 1)
	})

	// Sleep for a bit and ensure the value has increased.
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&i) == 0 {
		t.Error("Counter failed to increment with ticker")
	}

	cancel()

	// Sleep for a bit to let the cancel take place.
	time.Sleep(100 * time.Millisecond)

	last := atomic.LoadInt32(&i)

	// Sleep for a bit and ensure the value has not increased.
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&i) != last {
		t.Error("Counter incremented after stop")
	}
}

func TestEveryRunsWithTickerAndInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	params.SetupTestConfigCleanup(t)
	newCfg := params.BeaconConfig()
	newCfg.SecondsPerSlot = 1
	params.OverrideBeaconConfig(newCfg)

	i := int32(0)

	genTime := time.Now()
	async.RunWithTickerAndInterval(ctx, genTime, []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}, func() {
		atomic.AddInt32(&i, 1)
	})

	// Sleep for a bit and ensure the value has increased.
	time.Sleep(1150 * time.Millisecond)

	if atomic.LoadInt32(&i) != 1 {
		t.Errorf("Counter failed to increment with ticker: Got %d instead of %d", atomic.LoadInt32(&i), 1)
	}

	// Sleep for a bit and ensure the value has increased.
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&i) != 2 {
		t.Errorf("Counter failed to increment with ticker: Got %d instead of %d", atomic.LoadInt32(&i), 2)
	}

	cancel()

	// Sleep for a bit to let the cancel take place.
	time.Sleep(100 * time.Millisecond)

	last := atomic.LoadInt32(&i)

	// Sleep for a bit and ensure the value has not increased.
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&i) != last {
		t.Error("Counter incremented after stop")
	}
}
