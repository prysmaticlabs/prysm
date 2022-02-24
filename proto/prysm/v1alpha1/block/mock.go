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

func (m MockSignedBeaconBlock) Signature() []byte {
	panic("implement me")
}

func (m MockSignedBeaconBlock) IsNil() bool {
	return m.BeaconBlock == nil || m.Block().IsNil()
}

func (m MockSignedBeaconBlock) Copy() SignedBeaconBlock {
	panic("implement me")
}

func (m MockSignedBeaconBlock) Proto() proto.Message {
	panic("implement me")
}

func (m MockSignedBeaconBlock) PbPhase0Block() (*ethpb.SignedBeaconBlock, error) {
	panic("implement me")
}

func (m MockSignedBeaconBlock) PbAltairBlock() (*ethpb.SignedBeaconBlockAltair, error) {
	panic("implement me")
}

func (m MockSignedBeaconBlock) PbBellatrixBlock() (*ethpb.SignedBeaconBlockBellatrix, error) {
	panic("implement me")
}

func (m MockSignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	panic("implement me")
}

func (m MockSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	panic("implement me")
}

func (m MockSignedBeaconBlock) SizeSSZ() int {
	panic("implement me")
}

func (m MockSignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	panic("implement me")
}

func (m MockSignedBeaconBlock) Version() int {
	panic("implement me")
}

func (m MockSignedBeaconBlock) Header() (*ethpb.SignedBeaconBlockHeader, error) {
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

func (_ MockBeaconBlock) ProposerIndex() types.ValidatorIndex {
	panic("implement me")
}

func (_ MockBeaconBlock) ParentRoot() []byte {
	panic("implement me")
}

func (_ MockBeaconBlock) StateRoot() []byte {
	panic("implement me")
}

func (m MockBeaconBlock) Body() BeaconBlockBody {
	return m.BeaconBlockBody
}

func (_ MockBeaconBlock) IsNil() bool {
	return false
}

func (_ MockBeaconBlock) Proto() proto.Message {
	panic("implement me")
}

func (_ MockBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	panic("implement me")
}

func (_ MockBeaconBlock) MarshalSSZ() ([]byte, error) {
	panic("implement me")
}

func (_ MockBeaconBlock) SizeSSZ() int {
	panic("implement me")
}

func (_ MockBeaconBlock) UnmarshalSSZ(buf []byte) error {
	panic("implement me")
}

func (_ MockBeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	panic("implement me")
}

func (_ MockBeaconBlock) Version() int {
	panic("implement me")
}

type MockBeaconBlockBody struct{}

func (_ MockBeaconBlockBody) RandaoReveal() []byte {
	panic("implement me")
}

func (_ MockBeaconBlockBody) Eth1Data() *ethpb.Eth1Data {
	panic("implement me")
}

func (_ MockBeaconBlockBody) Graffiti() []byte {
	panic("implement me")
}

func (_ MockBeaconBlockBody) ProposerSlashings() []*ethpb.ProposerSlashing {
	panic("implement me")
}

func (_ MockBeaconBlockBody) AttesterSlashings() []*ethpb.AttesterSlashing {
	panic("implement me")
}

func (_ MockBeaconBlockBody) Attestations() []*ethpb.Attestation {
	panic("implement me")
}

func (_ MockBeaconBlockBody) Deposits() []*ethpb.Deposit {
	panic("implement me")
}

func (_ MockBeaconBlockBody) VoluntaryExits() []*ethpb.SignedVoluntaryExit {
	panic("implement me")
}

func (_ MockBeaconBlockBody) SyncAggregate() (*ethpb.SyncAggregate, error) {
	panic("implement me")
}

func (_ MockBeaconBlockBody) IsNil() bool {
	return false
}

func (_ MockBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	panic("implement me")
}

func (_ MockBeaconBlockBody) Proto() proto.Message {
	panic("implement me")
}

func (_ MockBeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	panic("implement me")
}

var _ SignedBeaconBlock = &MockSignedBeaconBlock{}
var _ BeaconBlock = &MockBeaconBlock{}
var _ BeaconBlockBody = &MockBeaconBlockBody{}
