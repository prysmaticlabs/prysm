package forkchoice

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func BenchmarkForkChoiceTree1(b *testing.B) {
	ctx := context.Background()
	db := testDB.SetupDB(b)
	defer testDB.TeardownDB(b, db)

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

	if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		b.Fatal(err)
	}

	store.justifiedCheckpt.Root = roots[0]
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		b.Fatal(err)
	}

	if err := store.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: store.justifiedCheckpt,
		State:      s,
	}); err != nil {
		b.Fatal(err)
	}

	// Spread out the votes evenly for all 3 leaf nodes
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 256:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[1]}); err != nil {
				b.Fatal(err)
			}
		case i > 768:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[7]}); err != nil {
				b.Fatal(err)
			}
		default:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[8]}); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Head(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkForkChoiceTree2(b *testing.B) {
	ctx := context.Background()
	db := testDB.SetupDB(b)
	defer testDB.TeardownDB(b, db)

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
	if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		b.Fatal(err)
	}

	store.justifiedCheckpt.Root = roots[0]
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		b.Fatal(err)
	}

	if err := store.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: store.justifiedCheckpt,
		State:      s,
	}); err != nil {
		b.Fatal(err)
	}

	// Spread out the votes evenly for all the leaf nodes. 8 to 15
	nodeIndex := 8
	for i := 0; i < len(validators); i++ {
		if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[nodeIndex]}); err != nil {
			b.Fatal(err)
		}
		if i%155 == 0 {
			nodeIndex++
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Head(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkForkChoiceTree3(b *testing.B) {
	ctx := context.Background()
	db := testDB.SetupDB(b)
	defer testDB.TeardownDB(b, db)

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
	if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		b.Fatal(err)
	}

	store.justifiedCheckpt.Root = roots[0]
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		b.Fatal(err)
	}

	if err := store.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: store.justifiedCheckpt,
		State:      s,
	}); err != nil {
		b.Fatal(err)
	}

	// All validators vote on the same head
	for i := 0; i < len(validators); i++ {
		if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[len(roots)-1]}); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Head(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
