package light

import (
	ssz "github.com/ferranbt/fastssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

// Precomputed values for generalized indices.
const (
	FinalizedRootIndex              = 105
	FinalizedRootIndexFloorLog2     = 6
	NextSyncCommitteeIndex          = 55
	NextSyncCommitteeIndexFloorLog2 = 5
)

type Service struct {
	prevHeadData map[[32]byte]*update
}

type update struct {
	header                  *ethpb.BeaconBlockHeader
	finalityCheckpoint      *ethpb.Checkpoint
	finalityBranch          *ssz.Proof
	nextSyncCommittee       *ethpb.SyncCommittee
	nextSyncCommitteeBranch *ssz.Proof
}

func (s *Service) onHead(head *ethpb.BeaconBlock, postState *ethpb.BeaconStateAltair) error {
	tr, err := postState.GetTree()
	if err != nil {
		return err
	}
	header, err := block.BeaconBlockHeaderFromBlock(head)
	if err != nil {
		return err
	}
	finalityBranch, err := tr.Prove(FinalizedRootIndex)
	if err != nil {
		return err
	}
	nextSyncCommitteeBranch, err := tr.Prove(NextSyncCommitteeIndex)
	if err != nil {
		return err
	}
	blkRoot, err := head.HashTreeRoot()
	if err != nil {
		return err
	}
	s.prevHeadData[blkRoot] = &update{
		header:                  header,
		finalityCheckpoint:      postState.FinalizedCheckpoint,
		finalityBranch:          finalityBranch,
		nextSyncCommittee:       postState.NextSyncCommittee,
		nextSyncCommitteeBranch: nextSyncCommitteeBranch,
	}
	return nil
}
