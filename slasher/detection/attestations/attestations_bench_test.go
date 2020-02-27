package attestations

import (
	"context"
	"fmt"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
)

func BenchmarkMinSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	db := testDB.SetupSlasherDB(b, true)
	defer testDB.TeardownSlasherDB(b, db)
	ctx := context.Background()

	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := db.ValidatorSpansMap(ctx, i%10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = detectAndUpdateMinEpochSpan(ctx, spanMap, i, i+diff)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMaxSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	db := testDB.SetupSlasherDB(b, true)
	defer testDB.TeardownSlasherDB(b, db)
	ctx := context.Background()

	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := db.ValidatorSpansMap(ctx, i%10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = detectAndUpdateMaxEpochSpan(ctx, spanMap, diff, diff+i)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDetectSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	db := testDB.SetupSlasherDB(b, true)
	defer testDB.TeardownSlasherDB(b, db)
	ctx := context.Background()

	for _, diff := range diffs {
		b.Run(fmt.Sprintf("Detect_MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := db.ValidatorSpansMap(ctx, i%10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, _, err = detectSlashingByEpochSpan(ctx, spanMap, i, i+diff, detectMax)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("Detect_MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := db.ValidatorSpansMap(ctx, i%10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, _, err = detectSlashingByEpochSpan(ctx, spanMap, i, i+diff, detectMin)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
