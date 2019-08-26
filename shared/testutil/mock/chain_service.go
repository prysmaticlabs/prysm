package mock

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

type ChainService struct {
	State               *pb.BeaconState
	Root                []byte
	FinalizedCheckPoint *ethpb.Checkpoint
}

func (ms *ChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *ChainService) ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *ChainService) ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *ChainService) HeadSlot() uint64 {
	return ms.State.Slot

}
func (ms *ChainService) HeadRoot() []byte {
	return ms.Root

}
func (ms *ChainService) HeadBlock() *ethpb.BeaconBlock {
	return nil
}

func (ms *ChainService) HeadState() *pb.BeaconState {
	return ms.State
}

func (ms *ChainService) FinalizedCheckpt() *ethpb.Checkpoint {
	return ms.FinalizedCheckPoint
}

func (ms *ChainService) ReceiveAttestation(context.Context, *ethpb.Attestation) error {
	return nil
}

func (ms *ChainService) ReceiveAttestationNoPubsub(context.Context, *ethpb.Attestation) error {
	return nil
}
