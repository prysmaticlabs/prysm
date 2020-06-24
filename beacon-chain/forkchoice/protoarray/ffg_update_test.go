package protoarray

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestFFGUpdates_OneBranch(t *testing.T) {
	balances := []uint64{1, 1}
	f := setup(0, 0)

	// The head should always start at the finalized block.
	r, err := f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != params.BeaconConfig().ZeroHash {
		t.Errorf("Incorrect head with genesis")
	}

	// Define the following tree:
	//            0 <- justified: 0, finalized: 0
	//            |
	//            1 <- justified: 0, finalized: 0
	//            |
	//            2 <- justified: 1, finalized: 0
	//            |
	//            3 <- justified: 2, finalized: 1
	if err := f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, [32]byte{}, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 2, indexToHash(2), indexToHash(1), [32]byte{}, 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 3, indexToHash(3), indexToHash(2), [32]byte{}, 2, 1); err != nil {
		t.Fatal(err)
	}

	// With starting justified epoch at 0, the head should be 3:
	//            0 <- start
	//            |
	//            1
	//            |
	//            2
	//            |
	//            3 <- head
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(3) {
		t.Error("Incorrect head for with justified epoch at 0")
	}

	// With starting justified epoch at 1, the head should be 2:
	//            0
	//            |
	//            1 <- start
	//            |
	//            2 <- head
	//            |
	//            3
	r, err = f.Head(context.Background(), 1, indexToHash(2), balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(2) {
		t.Error("Incorrect head with justified epoch at 1")
	}

	// With starting justified epoch at 2, the head should be 3:
	//            0
	//            |
	//            1
	//            |
	//            2 <- start
	//            |
	//            3 <- head
	r, err = f.Head(context.Background(), 2, indexToHash(3), balances, 1)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(3) {
		t.Error("Incorrect head with justified epoch at 2")
	}
}

func TestFFGUpdates_TwoBranches(t *testing.T) {
	balances := []uint64{1, 1}
	f := setup(0, 0)

	r, err := f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != params.BeaconConfig().ZeroHash {
		t.Errorf("Incorrect head with genesis")
	}

	// Define the following tree:
	//                                0
	//                               / \
	//  justified: 0, finalized: 0 -> 1   2 <- justified: 0, finalized: 0
	//                              |   |
	//  justified: 1, finalized: 0 -> 3   4 <- justified: 0, finalized: 0
	//                              |   |
	//  justified: 1, finalized: 0 -> 5   6 <- justified: 0, finalized: 0
	//                              |   |
	//  justified: 1, finalized: 0 -> 7   8 <- justified: 1, finalized: 0
	//                              |   |
	//  justified: 2, finalized: 0 -> 9  10 <- justified: 2, finalized: 0
	// Left branch.
	if err := f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, [32]byte{}, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 2, indexToHash(3), indexToHash(1), [32]byte{}, 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 3, indexToHash(5), indexToHash(3), [32]byte{}, 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 4, indexToHash(7), indexToHash(5), [32]byte{}, 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 4, indexToHash(9), indexToHash(7), [32]byte{}, 2, 0); err != nil {
		t.Fatal(err)
	}
	// Right branch.
	if err := f.ProcessBlock(context.Background(), 1, indexToHash(2), params.BeaconConfig().ZeroHash, [32]byte{}, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 2, indexToHash(4), indexToHash(2), [32]byte{}, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 3, indexToHash(6), indexToHash(4), [32]byte{}, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 4, indexToHash(8), indexToHash(6), [32]byte{}, 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(context.Background(), 4, indexToHash(10), indexToHash(8), [32]byte{}, 2, 0); err != nil {
		t.Fatal(err)
	}

	// With start at 0, the head should be 10:
	//           0  <-- start
	//          / \
	//         1   2
	//         |   |
	//         3   4
	//         |   |
	//         5   6
	//         |   |
	//         7   8
	//         |   |
	//         9  10 <-- head
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(10) {
		t.Error("Incorrect head with justified epoch at 0")
	}

	// Add a vote to 1:
	//                 0
	//                / \
	//    +1 vote -> 1   2
	//               |   |
	//               3   4
	//               |   |
	//               5   6
	//               |   |
	//               7   8
	//               |   |
	//               9  10
	f.ProcessAttestation(context.Background(), []uint64{0}, indexToHash(1), 0)

	// With the additional vote to the left branch, the head should be 9:
	//           0  <-- start
	//          / \
	//         1   2
	//         |   |
	//         3   4
	//         |   |
	//         5   6
	//         |   |
	//         7   8
	//         |   |
	// head -> 9  10
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(9) {
		t.Error("Incorrect head with justified epoch at 0")
	}

	// Add a vote to 2:
	//                 0
	//                / \
	//               1   2 <- +1 vote
	//               |   |
	//               3   4
	//               |   |
	//               5   6
	//               |   |
	//               7   8
	//               |   |
	//               9  10
	f.ProcessAttestation(context.Background(), []uint64{1}, indexToHash(2), 0)

	// With the additional vote to the right branch, the head should be 10:
	//           0  <-- start
	//          / \
	//         1   2
	//         |   |
	//         3   4
	//         |   |
	//         5   6
	//         |   |
	//         7   8
	//         |   |
	//         9  10 <-- head
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(10) {
		t.Error("Incorrect head with justified epoch at 0")
	}

	r, err = f.Head(context.Background(), 1, indexToHash(1), balances, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(7) {
		t.Error("Incorrect head with justified epoch at 0")
	}
}

func setup(justifiedEpoch uint64, finalizedEpoch uint64) *ForkChoice {
	f := New(0, 0, params.BeaconConfig().ZeroHash)
	f.store.NodeIndices[params.BeaconConfig().ZeroHash] = 0
	f.store.Nodes = append(f.store.Nodes, &Node{
		Slot:           0,
		Root:           params.BeaconConfig().ZeroHash,
		Parent:         NonExistentNode,
		JustifiedEpoch: justifiedEpoch,
		FinalizedEpoch: finalizedEpoch,
		BestChild:      NonExistentNode,
		BestDescendant: NonExistentNode,
		Weight:         0,
	})

	return f
}
