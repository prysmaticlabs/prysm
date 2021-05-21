package interfaces

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
)

type WrappedSignedBeaconBlock struct {
	b *ethpb.SignedBeaconBlock
}

func NewWrappedSignedBeaconBlock(b *ethpb.SignedBeaconBlock) WrappedSignedBeaconBlock {
	return WrappedSignedBeaconBlock{b: b}
}

func (w WrappedSignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

func (w WrappedSignedBeaconBlock) Block() BeaconBlock {
	return NewWrappedBeaconBlock(w.b.Block)
}

func (w WrappedSignedBeaconBlock) IsNil() bool {
	return w.b == nil
}

func (w WrappedSignedBeaconBlock) Copy() WrappedSignedBeaconBlock {
	return NewWrappedSignedBeaconBlock(blockutil.CopySignedBeaconBlock(w.b))
}

type WrappedBeaconBlock struct {
	b *ethpb.BeaconBlock
}

func NewWrappedBeaconBlock(b *ethpb.BeaconBlock) WrappedBeaconBlock {
	return WrappedBeaconBlock{b: b}
}

func (w WrappedBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

func (w WrappedBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

func (w WrappedBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

func (w WrappedBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

func (w WrappedBeaconBlock) Body() BeaconBlockBody {
	return NewWrappedBeaconBlockBody(w.b.Body)
}

func (w WrappedBeaconBlock) IsNil() bool {
	return w.b == nil
}

func (w WrappedBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

type WrappedBeaconBlockBody struct {
	b *ethpb.BeaconBlockBody
}

func NewWrappedBeaconBlockBody(b *ethpb.BeaconBlockBody) WrappedBeaconBlockBody {
	return WrappedBeaconBlockBody{b: b}
}

func (w WrappedBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

func (w WrappedBeaconBlockBody) Eth1Data() *ethpb.Eth1Data {
	return w.b.Eth1Data
}

func (w WrappedBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

func (w WrappedBeaconBlockBody) ProposerSlashings() []*ethpb.ProposerSlashing {
	return w.b.ProposerSlashings
}

func (w WrappedBeaconBlockBody) AttesterSlashings() []*ethpb.AttesterSlashing {
	return w.b.AttesterSlashings
}

func (w WrappedBeaconBlockBody) Attestations() []*ethpb.Attestation {
	return w.b.Attestations
}

func (w WrappedBeaconBlockBody) Deposits() []*ethpb.Deposit {
	return w.b.Deposits
}

func (w WrappedBeaconBlockBody) VoluntaryExits() []*ethpb.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

func (w WrappedBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

func (w WrappedBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}
