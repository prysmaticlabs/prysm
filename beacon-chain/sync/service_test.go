package sync

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

type mockChainService struct {
	headState        *pb.BeaconState
	headRoot         []byte
	finalizedCheckpt *ethpb.Checkpoint
}

func (ms *mockChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *mockChainService) ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *mockChainService) ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *mockChainService) HeadSlot() uint64 {
	return ms.headState.Slot

}
func (ms *mockChainService) HeadRoot() []byte {
	return ms.headRoot

}
func (ms *mockChainService) HeadBlock() *ethpb.BeaconBlock {
	return nil
}

func (ms *mockChainService) HeadState() *pb.BeaconState {
	return ms.headState
}

func (ms *mockChainService) FinalizedCheckpt() *ethpb.Checkpoint {
	return ms.finalizedCheckpt
}

func (ms *mockChainService) ReceiveAttestation(context.Context, *ethpb.Attestation) error {
	return nil
}

func (ms *mockChainService) ReceiveAttestationNoPubsub(context.Context, *ethpb.Attestation) error {
	return nil
}
