package forkchoice

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func BenchmarkForkChoiceTree1(b *testing.B) {
	ctx := context.Background()
	db := internal.SetupDB(b)
	defer internal.TeardownDB(b, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db)
	if err != nil {
		b.Fatal(err)
	}

	// Benchmark fork choice with 1024 validators
	validators := make([]*ethpb.Validator, 1024)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators}
	if err := store.GensisStore(s); err != nil {
		b.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	if err := store.db.SaveCheckpointState(ctx, s, store.justifiedCheckpt); err != nil {
		b.Fatal(err)
	}

	// Spread out the votes evenly for all 3 leaf nodes
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 256:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[1]}); err != nil {
				b.Fatal(err)
			}
		case i > 768:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[7]}); err != nil {
				b.Fatal(err)
			}
		default:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[8]}); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Head()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkForkChoiceTree2(b *testing.B) {
	ctx := context.Background()
	db := internal.SetupDB(b)
	defer internal.TeardownDB(b, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree2(db)
	if err != nil {
		b.Fatal(err)
	}

	// Benchmark fork choice with 1024 validators
	validators := make([]*ethpb.Validator, 1024)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators}
	if err := store.GensisStore(s); err != nil {
		b.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	if err := store.db.SaveCheckpointState(ctx, s, store.justifiedCheckpt); err != nil {
		b.Fatal(err)
	}

	// Spread out the votes evenly for all the leaf nodes. 8 to 15
	nodeIndex := 8
	for i := 0; i < len(validators); i++ {
		if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[nodeIndex]}); err != nil {
			b.Fatal(err)
		}
		if i%155 == 0 {
			nodeIndex++
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Head()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkForkChoiceTree3(b *testing.B) {
	ctx := context.Background()
	db := internal.SetupDB(b)
	defer internal.TeardownDB(b, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree3(db)
	if err != nil {
		b.Fatal(err)
	}

	// Benchmark fork choice with 1024 validators
	validators := make([]*ethpb.Validator, 1024)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators}
	if err := store.GensisStore(s); err != nil {
		b.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	if err := store.db.SaveCheckpointState(ctx, s, store.justifiedCheckpt); err != nil {
		b.Fatal(err)
	}

	// All validators vote on the same head
	for i := 0; i < len(validators); i++ {
		if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[len(roots)-1]}); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Head()
		if err != nil {
			b.Fatal(err)
		}
	}
}
