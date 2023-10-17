package doublylinkedtree

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func Benchmark_OnBlock(b *testing.B) {
	steps := []int{100, 1000, 10000, 20000, 40000}

	for i := range steps {
		b.Run(fmt.Sprintf("non_finalizing_epochs_%d", steps[i]), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				b.StopTimer()
				f := setup(0, 0)
				ctx := context.Background()
				parentRoot := [32]byte{}
				newRoot := [32]byte{}
				for epoch := primitives.Epoch(0); epoch < primitives.Epoch(steps[i]); epoch++ {
					slot, err := slots.EpochStart(epoch)
					require.NoError(b, err)
					_, err = rand.Read(newRoot[:])
					require.NoError(b, err)
					st, root, err := prepareForkchoiceState(ctx, slot, newRoot, parentRoot, [32]byte{}, 0, 0)
					require.NoError(b, err)
					require.NoError(b, f.InsertNode(ctx, st, root))
					require.NoError(b, f.SetOptimisticToValid(ctx, newRoot))
					parentRoot = newRoot
				}
				b.StartTimer()
				root, err := f.Head(ctx)
				require.NoError(b, err)
				require.Equal(b, newRoot, root)
			}
		})
	}
}
