package rpc

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
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
				spanMap, err := slasherServer.SlasherDB.ValidatorSpansMap(i % 10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = slasherServer.DetectAndUpdateMinEpochSpan(ctx, i, i+diff, i%10, spanMap)
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
				spanMap, err := slasherServer.SlasherDB.ValidatorSpansMap(i % 10)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = slasherServer.DetectAndUpdateMaxEpochSpan(ctx, diff, diff+i, i%10, spanMap)
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
	dbs := db.SetupSlasherDB(b)
	defer db.TeardownSlasherDB(b, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	var cb []uint64
	for i := uint64(0); i < 100; i++ {
		cb = append(cb, i)
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: cb,
		CustodyBit_1Indices: []uint64{},
		Signature:           make([]byte, 96),
		Data: &ethpb.AttestationData{
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	for i := uint64(0); i < uint64(b.N); i++ {
		ia1.Data.Target.Epoch = i + 1
		ia1.Data.Source.Epoch = i
		b.Logf("In Loop: %d", i)
		ia1.Data.Slot = (i + 1) * params.BeaconConfig().SlotsPerEpoch
		root := []byte(strconv.Itoa(int(i)))
		ia1.Data.BeaconBlockRoot = append(root, ia1.Data.BeaconBlockRoot[len(root):]...)
		if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
			b.Errorf("Could not call RPC method: %v", err)
		}
	}

	s, err := dbs.Size()
	if err != nil {
		b.Error(err)
	}
	b.Logf("DB size is: %d", s)

}
