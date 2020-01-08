package rpc

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/flags"

	"github.com/urfave/cli"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/db"
)

var appFlags = []cli.Flag{
	flags.CertFlag,
	flags.RPCPort,
	flags.KeyFlag,
	flags.UseSpanCacheFlag,
}

func BenchmarkMinSpan(b *testing.B) {
	diffs := []uint64{2, 10, 100, 1000, 10000, 53999}
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	ctx := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(b, ctx)
	defer db.TeardownSlasherDB(b, dbs)

	context := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := slasherServer.SlasherDB.ValidatorSpansMap(i % 10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = slasherServer.DetectAndUpdateMinEpochSpan(context, i, i+diff, i%10, spanMap)
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
	ctx := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(b, ctx)
	defer db.TeardownSlasherDB(b, dbs)

	context := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := slasherServer.SlasherDB.ValidatorSpansMap(i % 10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = slasherServer.DetectAndUpdateMaxEpochSpan(context, diff, diff+i, i%10, spanMap)
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
	ctx := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(b, ctx)
	defer db.TeardownSlasherDB(b, dbs)

	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("Detect_MaxSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := slasherServer.SlasherDB.ValidatorSpansMap(i % 10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, _, err = slasherServer.detectSlashingByEpochSpan(i, i+diff, spanMap, detectMax)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	for _, diff := range diffs {
		b.Run(fmt.Sprintf("Detect_MinSpan_diff_%d", diff), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				spanMap, err := slasherServer.SlasherDB.ValidatorSpansMap(i % 10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, _, err = slasherServer.detectSlashingByEpochSpan(i, i+diff, spanMap, detectMin)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCheckAttestations(b *testing.B) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	ctx := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(b, ctx)
	defer db.TeardownSlasherDB(b, dbs)
	context := context.Background()
	slasherServer := &Server{
		ctx:       context,
		SlasherDB: dbs,
	}
	var cb []uint64
	for i := uint64(0); i < 100; i++ {
		cb = append(cb, i)
	}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: cb,
		Signature:        make([]byte, 96),
		Data: &ethpb.AttestationData{
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	b.ResetTimer()
	for i := uint64(0); i < uint64(b.N); i++ {
		ia1.Data.Target.Epoch = i + 1
		ia1.Data.Source.Epoch = i
		ia1.Data.Slot = (i + 1) * params.BeaconConfig().SlotsPerEpoch
		root := []byte(strconv.Itoa(int(i)))
		ia1.Data.BeaconBlockRoot = append(root, ia1.Data.BeaconBlockRoot[len(root):]...)
		if _, err := slasherServer.IsSlashableAttestation(context, ia1); err != nil {
			b.Errorf("Could not call RPC method: %v", err)
		}
	}

}
