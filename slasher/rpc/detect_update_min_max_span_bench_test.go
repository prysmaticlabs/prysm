package rpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/db"
)

func BenchmarkMinSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	dbs := db.SetupSlasherDB(b)
	defer db.TeardownSlasherDB(b, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MinSpan with diff: %d", diff), func(ib *testing.B) {
			for i := uint64(ib.N) - 10; i < uint64(ib.N); i++ {
				_, _ = slasherServer.DetectAndUpdateMinSpan(ctx, i, i+diff, 1)

			}
		})
	}
}
