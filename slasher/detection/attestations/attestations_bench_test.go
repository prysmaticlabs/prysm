package attestations

import (
	"context"
	"flag"
	"fmt"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/urfave/cli"
)

func BenchmarkMinSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	db := testDB.SetupSlasherDB(b, cli.NewContext(app, set, nil))
	defer testDB.TeardownSlasherDB(b, db)
	ctx := context.Background()

	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := db.ValidatorSpansMap(ctx, i%10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = detectAndUpdateMinEpochSpan(ctx, i, i+diff, spanMap)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMaxSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	db := testDB.SetupSlasherDB(b, cli.NewContext(app, set, nil))
	defer testDB.TeardownSlasherDB(b, db)
	ctx := context.Background()

	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := db.ValidatorSpansMap(ctx, i%10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = detectAndUpdateMaxEpochSpan(ctx, diff, diff+i, spanMap)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDetectSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	db := testDB.SetupSlasherDB(b, cli.NewContext(app, set, nil))
	defer testDB.TeardownSlasherDB(b, db)
	ctx := context.Background()

	for _, diff := range diffs {
		b.Run(fmt.Sprintf("Detect_MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := db.ValidatorSpansMap(ctx, i%10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, _, err = detectSlashingByEpochSpan(ctx, i, i+diff, spanMap, detectMax)
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
				_, _, _, err = detectSlashingByEpochSpan(ctx, i, i+diff, spanMap, detectMin)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
