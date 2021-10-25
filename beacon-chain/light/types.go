package light

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type ClientSnapshot struct {
	Header               *ethpb.BeaconBlockHeader
	CurrentSyncCommittee *ethpb.SyncCommittee
	NextSyncCommittee    *ethpb.SyncCommittee
}

type ClientUpdate struct {
	Header                  *ethpb.BeaconBlockHeader
	NextSyncCommittee       *ethpb.SyncCommittee
	NextSyncCommitteeBranch [][32]byte
	FinalityHeader          *ethpb.BeaconBlockHeader
	FinalityBranch          [][32]byte
	SyncCommitteeBits       bitfield.Bitvector512
	SyncCommitteeSignature  [96]byte
	ForkVersion             [4]byte
}

type ClientStore struct {
	Snapshot     *ClientSnapshot
	ValidUpdates []*ClientUpdate
}

type SyncAttestedData struct {
	Header                  *ethpb.BeaconBlockHeader
	FinalityCheckpoint      *ethpb.Checkpoint
	FinalityBranch          *ssz.Proof
	NextSyncCommittee       *ethpb.SyncCommittee
	NextSyncCommitteeBranch *ssz.Proof
}
