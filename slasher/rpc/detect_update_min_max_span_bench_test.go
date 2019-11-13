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
		b.Run(fmt.Sprintf("MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				_, err := slasherServer.DetectAndUpdateMinSpan(ctx, i, i+diff, i%10)
				if err != nil {
					b.Fatal(err)
				}
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
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				_, err := slasherServer.DetectAndUpdateMaxSpan(ctx, diff, diff+i, i%10)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDetectSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	dbs := db.SetupSlasherDB(b)
	defer db.TeardownSlasherDB(b, dbs)

	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("Detect_MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				_, _, _, err := slasherServer.detectSlashingByEpochSpan(i, i+diff, i%10, detectEpochMaxSpan)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("Detect_MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				_, _, _, err := slasherServer.detectSlashingByEpochSpan(i, i+diff, i%10, detectEpochMinSpan)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
