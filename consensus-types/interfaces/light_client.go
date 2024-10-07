package interfaces

import (
	ssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

const (
	ExecutionBranchNumOfLeaves     = 4
	SyncCommitteeBranchNumOfLeaves = 5
	FinalityBranchNumOfLeaves      = 6
)

type LightClientExecutionBranch = [ExecutionBranchNumOfLeaves][fieldparams.RootLength]byte
type LightClientSyncCommitteeBranch = [SyncCommitteeBranchNumOfLeaves][fieldparams.RootLength]byte
type LightClientFinalityBranch = [FinalityBranchNumOfLeaves][fieldparams.RootLength]byte

type LightClientHeader interface {
	ssz.Marshaler
	Version() int
	Beacon() *pb.BeaconBlockHeader
	Execution() (ExecutionData, error)
	ExecutionBranch() (LightClientExecutionBranch, error)
}

type LightClientBootstrap interface {
	ssz.Marshaler
	Version() int
	Header() LightClientHeader
	CurrentSyncCommittee() *pb.SyncCommittee
	CurrentSyncCommitteeBranch() LightClientSyncCommitteeBranch
}

type LightClientUpdate interface {
	ssz.Marshaler
	Version() int
	AttestedHeader() LightClientHeader
	NextSyncCommittee() *pb.SyncCommittee
	NextSyncCommitteeBranch() LightClientSyncCommitteeBranch
	FinalizedHeader() LightClientHeader
	FinalityBranch() LightClientFinalityBranch
	SyncAggregate() *pb.SyncAggregate
	SignatureSlot() primitives.Slot
}

type LightClientFinalityUpdate interface {
	ssz.Marshaler
	Version() int
	AttestedHeader() LightClientHeader
	FinalizedHeader() LightClientHeader
	FinalityBranch() LightClientFinalityBranch
	SyncAggregate() *pb.SyncAggregate
	SignatureSlot() primitives.Slot
}

type LightClientOptimisticUpdate interface {
	ssz.Marshaler
	Version() int
	AttestedHeader() LightClientHeader
	SyncAggregate() *pb.SyncAggregate
	SignatureSlot() primitives.Slot
}
