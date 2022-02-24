package block

import (
	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

type MockSignedBeaconBlock struct {
	BeaconBlock BeaconBlock
}

func (m MockSignedBeaconBlock) Block() BeaconBlock {
	return m.BeaconBlock
}

func (MockSignedBeaconBlock) Signature() []byte {
	panic("implement me")
}

func (m MockSignedBeaconBlock) IsNil() bool {
	return m.BeaconBlock == nil || m.Block().IsNil()
}

func (MockSignedBeaconBlock) Copy() SignedBeaconBlock {
	panic("implement me")
}

func (MockSignedBeaconBlock) Proto() proto.Message {
	panic("implement me")
}

func (MockSignedBeaconBlock) PbPhase0Block() (*ethpb.SignedBeaconBlock, error) {
	panic("implement me")
}

func (MockSignedBeaconBlock) PbAltairBlock() (*ethpb.SignedBeaconBlockAltair, error) {
	panic("implement me")
}

func (MockSignedBeaconBlock) PbBellatrixBlock() (*ethpb.SignedBeaconBlockBellatrix, error) {
	panic("implement me")
}

func (MockSignedBeaconBlock) MarshalSSZTo(_ []byte) ([]byte, error) {
	panic("implement me")
}

func (MockSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	panic("implement me")
}

func (MockSignedBeaconBlock) SizeSSZ() int {
	panic("implement me")
}

func (MockSignedBeaconBlock) UnmarshalSSZ(_ []byte) error {
	panic("implement me")
}

func (MockSignedBeaconBlock) Version() int {
	panic("implement me")
}

func (MockSignedBeaconBlock) Header() (*ethpb.SignedBeaconBlockHeader, error) {
	panic("implement me")
}

type MockBeaconBlock struct {
	Htr             [32]byte
	HtrErr          error
	BeaconBlockBody BeaconBlockBody
	BlockSlot       types.Slot
}

func (m MockBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return m.Htr, m.HtrErr
}

func (m MockBeaconBlock) Slot() types.Slot {
	return m.BlockSlot
}

func (MockBeaconBlock) ProposerIndex() types.ValidatorIndex {
	panic("implement me")
}

func (MockBeaconBlock) ParentRoot() []byte {
	panic("implement me")
}

func (MockBeaconBlock) StateRoot() []byte {
	panic("implement me")
}

func (m MockBeaconBlock) Body() BeaconBlockBody {
	return m.BeaconBlockBody
}

func (MockBeaconBlock) IsNil() bool {
	return false
}

func (MockBeaconBlock) Proto() proto.Message {
	panic("implement me")
}

func (MockBeaconBlock) MarshalSSZTo(_ []byte) ([]byte, error) {
	panic("implement me")
}

func (MockBeaconBlock) MarshalSSZ() ([]byte, error) {
	panic("implement me")
}

func (MockBeaconBlock) SizeSSZ() int {
	panic("implement me")
}

func (MockBeaconBlock) UnmarshalSSZ(_ []byte) error {
	panic("implement me")
}

func (MockBeaconBlock) HashTreeRootWith(_ *ssz.Hasher) error {
	panic("implement me")
}

func (MockBeaconBlock) Version() int {
	panic("implement me")
}

type MockBeaconBlockBody struct{}

func (MockBeaconBlockBody) RandaoReveal() []byte {
	panic("implement me")
}

func (MockBeaconBlockBody) Eth1Data() *ethpb.Eth1Data {
	panic("implement me")
}

func (MockBeaconBlockBody) Graffiti() []byte {
	panic("implement me")
}

func (MockBeaconBlockBody) ProposerSlashings() []*ethpb.ProposerSlashing {
	panic("implement me")
}

func (MockBeaconBlockBody) AttesterSlashings() []*ethpb.AttesterSlashing {
	panic("implement me")
}

func (MockBeaconBlockBody) Attestations() []*ethpb.Attestation {
	panic("implement me")
}

func (MockBeaconBlockBody) Deposits() []*ethpb.Deposit {
	panic("implement me")
}

func (MockBeaconBlockBody) VoluntaryExits() []*ethpb.SignedVoluntaryExit {
	panic("implement me")
}

func (MockBeaconBlockBody) SyncAggregate() (*ethpb.SyncAggregate, error) {
	panic("implement me")
}

func (MockBeaconBlockBody) IsNil() bool {
	return false
}

func (MockBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	panic("implement me")
}

func (MockBeaconBlockBody) Proto() proto.Message {
	panic("implement me")
}

func (MockBeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	panic("implement me")
}

var _ SignedBeaconBlock = &MockSignedBeaconBlock{}
var _ BeaconBlock = &MockBeaconBlock{}
var _ BeaconBlockBody = &MockBeaconBlockBody{}
