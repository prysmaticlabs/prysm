package interfaces

import (
	types "github.com/prysmaticlabs/eth2-types"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

type SignedBeaconBlock interface {
	GetSignature() []byte
	GetBlock() *pb.BeaconBlock
}

type BeaconBlock interface {
	GetSlot() types.Slot
	GetProposerIndex() types.ValidatorIndex
	GetParentRoot() []byte
	GetStateRoot() []byte
	GetBody() *pb.BeaconBlockBody
	HashTreeRoot() ([32]byte, error)
}

type BeaconBlockBody interface {
	GetRandaoReveal() []byte
	GetEth1Data() *pb.Eth1Data
	GetProposerSlashings() []*pb.ProposerSlashing
	GetAttesterSlashings() []*pb.AttesterSlashing
	GetAttestations() []*pb.Attestation
	GetDeposits() []*pb.Deposit
	GetVoluntaryExits() []*pb.SignedVoluntaryExit
	HashTreeRoot() ([32]byte, error)
}
