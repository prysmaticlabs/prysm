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
	slasherServer.DetectAndUpdateMinSpan(ctx, 1, 53999, 1)
	slasherServer.DetectAndUpdateMinSpan(ctx, 53998, 53999, 1)
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(ib.N%54000) - 10; i < uint64(ib.N%54000); i++ {
				slasherServer.DetectAndUpdateMinSpan(ctx, i, i+diff, 1)
			}
		})
	}
}

func BenchmarkMaxSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	dbs := db.SetupSlasherDB(b)
	defer db.TeardownSlasherDB(b, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	slasherServer.DetectAndUpdateMaxSpan(ctx, 1, 53999, 1)
	slasherServer.DetectAndUpdateMinSpan(ctx, 53998, 53999, 1)
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(ib.N%54000) - 10; i < uint64(ib.N%54000); i++ {
				slasherServer.DetectAndUpdateMaxSpan(ctx, i, i+diff, 1)

			}
		})
	}
}
