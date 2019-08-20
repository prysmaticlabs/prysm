package forkchoice

import (
	"context"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// blockTree1 constructs the following tree:
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
// (B1, and B3 are all from the same slots)
func blockTree1(db db.Database) ([][]byte, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: []byte{'g'}}
	r0, _ := ssz.SigningRoot(b0)
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, _ := ssz.SigningRoot(b1)
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r0[:]}
	r3, _ := ssz.SigningRoot(b3)
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r3[:]}
	r4, _ := ssz.SigningRoot(b4)
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r4[:]}
	r5, _ := ssz.SigningRoot(b5)
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r4[:]}
	r6, _ := ssz.SigningRoot(b6)
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r5[:]}
	r7, _ := ssz.SigningRoot(b7)
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r6[:]}
	r8, _ := ssz.SigningRoot(b8)
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b3, b4, b5, b6, b7, b8} {
		if err := db.SaveBlock(context.Background(), b); err != nil {
			return nil, err
		}
		if err := db.SaveState(context.Background(), &pb.BeaconState{}, bytesutil.ToBytes32(b.ParentRoot)); err != nil {
			return nil, err
		}
	}
	return [][]byte{r0[:], r1[:], nil, r3[:], r4[:], r5[:], r6[:], r7[:], r8[:]}, nil
}

// blockTree2 constructs the following tree:
// Scenario graph: shorturl.at/loyP6
//
//digraph G {
//    rankdir=LR;
//    node [shape="none"];
//
//    subgraph blocks {
//        rankdir=LR;
//        node [shape="box"];
//        a->b;
//        a->c;
//        b->d;
//        b->e;
//        c->f;
//        c->g;
//        d->h
//        d->i
//        d->j
//        d->k
//        h->l
//        h->m
//        g->n
//        g->o
//        e->p
//    }
//}
func blockTree2(db db.Database) ([][]byte, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: []byte{'g'}}
	r0, _ := ssz.SigningRoot(b0)
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, _ := ssz.SigningRoot(b1)
	b2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:]}
	r2, _ := ssz.SigningRoot(b2)
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r1[:]}
	r3, _ := ssz.SigningRoot(b3)
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r1[:]}
	r4, _ := ssz.SigningRoot(b4)
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r2[:]}
	r5, _ := ssz.SigningRoot(b5)
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r2[:]}
	r6, _ := ssz.SigningRoot(b6)
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r3[:]}
	r7, _ := ssz.SigningRoot(b7)
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r3[:]}
	r8, _ := ssz.SigningRoot(b8)
	b9 := &ethpb.BeaconBlock{Slot: 9, ParentRoot: r3[:]}
	r9, _ := ssz.SigningRoot(b9)
	b10 := &ethpb.BeaconBlock{Slot: 10, ParentRoot: r3[:]}
	r10, _ := ssz.SigningRoot(b10)
	b11 := &ethpb.BeaconBlock{Slot: 11, ParentRoot: r4[:]}
	r11, _ := ssz.SigningRoot(b11)
	b12 := &ethpb.BeaconBlock{Slot: 12, ParentRoot: r6[:]}
	r12, _ := ssz.SigningRoot(b12)
	b13 := &ethpb.BeaconBlock{Slot: 13, ParentRoot: r6[:]}
	r13, _ := ssz.SigningRoot(b13)
	b14 := &ethpb.BeaconBlock{Slot: 14, ParentRoot: r7[:]}
	r14, _ := ssz.SigningRoot(b14)
	b15 := &ethpb.BeaconBlock{Slot: 15, ParentRoot: r7[:]}
	r15, _ := ssz.SigningRoot(b15)
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14, b15} {
		if err := db.SaveBlock(context.Background(), b); err != nil {
			return nil, err
		}
		if err := db.SaveState(context.Background(), &pb.BeaconState{}, bytesutil.ToBytes32(b.ParentRoot)); err != nil {
			return nil, err
		}
	}
	return [][]byte{r0[:], r1[:], r2[:], r3[:], r4[:], r5[:], r6[:], r7[:], r8[:], r9[:], r10[:], r11[:], r12[:], r13[:], r14[:], r15[:]}, nil
}

// blockTree3 constructs a tree that is 512 blocks in a row.
// B0 - B1 - B2 - B3 - .... - B512
func blockTree3(db db.Database) ([][]byte, error) {
	blkCount := 512
	roots := make([][]byte, 0, blkCount)
	blks := make([]*ethpb.BeaconBlock, 0, blkCount)
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: []byte{'g'}}
	r0, _ := ssz.SigningRoot(b0)
	roots = append(roots, r0[:])
	blks = append(blks, b0)

	for i := 1; i < blkCount; i++ {
		b := &ethpb.BeaconBlock{Slot: uint64(i), ParentRoot: roots[len(roots)-1]}
		r, _ := ssz.SigningRoot(b)
		roots = append(roots, r[:])
		blks = append(blks, b)
	}

	for _, b := range blks {
		if err := db.SaveBlock(context.Background(), b); err != nil {
			return nil, err
		}
		if err := db.SaveState(context.Background(), &pb.BeaconState{}, bytesutil.ToBytes32(b.ParentRoot)); err != nil {
			return nil, err
		}
	}
	return roots, nil
}
