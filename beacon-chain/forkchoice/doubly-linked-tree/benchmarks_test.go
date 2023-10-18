package doublylinkedtree

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func Test_LinearBehavior(t *testing.T) {
	// steps := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 2000, 3000, 5000, 10000, 20000, 40000}
	steps := []int{100, 500, 1000, 5000, 10000, 20000, 40000}
	durationsAccumulated := make([]time.Duration, len(steps))
	durationsLastInsertion := make([]time.Duration, len(steps))

	ctx := context.Background()
	for i := range steps {
		fmt.Println("Starting test: ", steps[i])
		t.Run(fmt.Sprintf("non_finalizing_epochs_%d", steps[i]), func(tt *testing.T) {
			start := time.Now()
			f := setup(0, 0)
			parentRoot := [32]byte{}
			newRoot := [32]byte{}
			var slot primitives.Slot
			var err error
			for epoch := primitives.Epoch(0); epoch < primitives.Epoch(steps[i]); epoch++ {
				slot, err = slots.EpochStart(epoch)
				require.NoError(tt, err)
				_, err = rand.Read(newRoot[:])
				require.NoError(tt, err)
				st, root, err := prepareForkchoiceState(ctx, slot, newRoot, parentRoot, [32]byte{}, 0, 0)
				require.NoError(tt, err)
				require.NoError(tt, f.InsertNode(ctx, st, root))
				require.NoError(tt, f.SetOptimisticToValid(ctx, newRoot))
				parentRoot = newRoot
			}
			durationsAccumulated[i] = time.Since(start)
			root, err := f.Head(ctx)
			require.NoError(tt, err)
			require.Equal(tt, newRoot, root)
			durationsLastInsertion[i] = time.Since(start) - durationsAccumulated[i]
			// Check that the latencies are compatible with previous
			// latencies. We expect the total insertion and head
			// computation to be linear. Local tests show 7ms per
			// insertion and 40ns per epoch for head computation.
			// Numbers here are conservative and should fail on
			// quadratic behavior
			fmt.Println("Duration accumulated: ", durationsAccumulated[i], ". Duration last insertion: ", durationsLastInsertion[i])
			if i == 0 {
				return
			}
			for j := 0; j < i; j++ {
				require.Equal(tt, true, durationsAccumulated[i] < time.Duration(steps[i]*steps[i])*time.Millisecond)
				if steps[i] > 1000 {
					require.Equal(tt, true, durationsLastInsertion[i] < time.Duration(steps[i]*steps[i]/1000)*time.Microsecond)
				}
			}
		})
	}
}
