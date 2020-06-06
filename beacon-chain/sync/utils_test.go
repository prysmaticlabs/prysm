package sync

import (
	"math/rand"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

func TestSortedObj_SortBlocksRoots(t *testing.T) {
	source := rand.NewSource(33)
	randGen := rand.New(source)
	blks := []*ethpb.SignedBeaconBlock{}
	roots := [][32]byte{}
	randFunc := func() int64 {
		return randGen.Int63n(50)
	}

	for i := 0; i < 10; i++ {
		slot := uint64(randFunc())
		newBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot}}
		blks = append(blks, newBlk)
		root := bytesutil.ToBytes32(bytesutil.Bytes32(slot))
		roots = append(roots, root)
	}

	r := &Service{}

	newBlks, newRoots := r.sortBlocksAndRoots(blks, roots)

	previousSlot := uint64(0)
	for i, b := range newBlks {
		if b.Block.Slot < previousSlot {
			t.Errorf("Block list is not sorted as %d is smaller than previousSlot %d", b.Block.Slot, previousSlot)
		}
		if bytesutil.FromBytes8(newRoots[i][:]) != b.Block.Slot {
			t.Errorf("root doesn't match stored slot in block: wanted %d but got %d", b.Block.Slot, bytesutil.FromBytes8(newRoots[i][:]))
		}
		previousSlot = b.Block.Slot
	}
}

func TestValidateAggregatedTime_ValidatesCorrectly(t *testing.T) {
	const genesisOffset = 1200
	genTime := roughtime.Now().Add(-(genesisOffset * time.Second))
	currSlot := helpers.SlotsSince(genTime)
	invalidAttSlot := currSlot - params.BeaconNetworkConfig().AttestationPropagationSlotRange - 1
	err := validateAggregateAttTime(invalidAttSlot, genTime)
	if err == nil {
		t.Error("Expected attestation time to be invalid, but it was marked as valid")
	}
	timePerSlot := time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
	// adjusts the genesis time to allow for the clock disparity to
	// allow for the attestation to be valid.
	clockAllowance := params.BeaconNetworkConfig().MaximumGossipClockDisparity * 8 / 10
	newTime := genTime.Add(timePerSlot - clockAllowance)
	err = validateAggregateAttTime(invalidAttSlot, newTime)
	if err != nil {
		t.Errorf("Expected attestation time to be valid, but it was not: %v", err)
	}
	// re-determine the current slot
	currSlot = helpers.SlotsSince(genTime)
	err = validateAggregateAttTime(currSlot+1, genTime)
	if err == nil {
		t.Error("Expected attestation time to be invalid, but it was marked as valid")
	}
	err = validateAggregateAttTime(currSlot-10, genTime)
	if err != nil {
		t.Errorf("Expected attestation time to be valid, but it was not: %v", err)
	}
}
