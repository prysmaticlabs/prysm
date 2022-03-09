package mock

import (
	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"google.golang.org/protobuf/proto"
)

type SignedBeaconBlock struct {
	BeaconBlock block.BeaconBlock
}

func (SignedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	panic("implement me")
}

func (m SignedBeaconBlock) Block() block.BeaconBlock {
	return m.BeaconBlock
}

func (SignedBeaconBlock) Signature() []byte {
	panic("implement me")
}

func (m SignedBeaconBlock) IsNil() bool {
	return m.BeaconBlock == nil || m.Block().IsNil()
}

func (SignedBeaconBlock) Copy() block.SignedBeaconBlock {
	panic("implement me")
}

func (SignedBeaconBlock) Proto() proto.Message {
	panic("implement me")
}

func (SignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	panic("implement me")
}

func (SignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	panic("implement me")
}

func (SignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	panic("implement me")
}

func (SignedBeaconBlock) MarshalSSZTo(_ []byte) ([]byte, error) {
	panic("implement me")
}

func (SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	panic("implement me")
}

func (SignedBeaconBlock) SizeSSZ() int {
	panic("implement me")
}

func (SignedBeaconBlock) UnmarshalSSZ(_ []byte) error {
	panic("implement me")
}

func (SignedBeaconBlock) Version() int {
	panic("implement me")
}

func (SignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
	panic("implement me")
}

type BeaconBlock struct {
	Htr             [32]byte
	HtrErr          error
	BeaconBlockBody block.BeaconBlockBody
	BlockSlot       types.Slot
}

func (BeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	panic("implement me")
}

func (m BeaconBlock) HashTreeRoot() ([32]byte, error) {
	return m.Htr, m.HtrErr
}

func (m BeaconBlock) Slot() types.Slot {
	return m.BlockSlot
}

func (BeaconBlock) ProposerIndex() types.ValidatorIndex {
	panic("implement me")
}

func (BeaconBlock) ParentRoot() []byte {
	panic("implement me")
}

func (BeaconBlock) StateRoot() []byte {
	panic("implement me")
}

func (m BeaconBlock) Body() block.BeaconBlockBody {
	return m.BeaconBlockBody
}

func (BeaconBlock) IsNil() bool {
	return false
}

func (BeaconBlock) Proto() proto.Message {
	panic("implement me")
}

func (BeaconBlock) MarshalSSZTo(_ []byte) ([]byte, error) {
	panic("implement me")
}

func (BeaconBlock) MarshalSSZ() ([]byte, error) {
	panic("implement me")
}

func (BeaconBlock) SizeSSZ() int {
	panic("implement me")
}

func (BeaconBlock) UnmarshalSSZ(_ []byte) error {
	panic("implement me")
}

func (BeaconBlock) HashTreeRootWith(_ *ssz.Hasher) error {
	panic("implement me")
}

func (BeaconBlock) Version() int {
	panic("implement me")
}

type BeaconBlockBody struct{}

func (BeaconBlockBody) RandaoReveal() []byte {
	panic("implement me")
}

func (BeaconBlockBody) Eth1Data() *eth.Eth1Data {
	panic("implement me")
}

func (BeaconBlockBody) Graffiti() []byte {
	panic("implement me")
}

func (BeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	panic("implement me")
}

func (BeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	panic("implement me")
}

func (BeaconBlockBody) Attestations() []*eth.Attestation {
	panic("implement me")
}

func (BeaconBlockBody) Deposits() []*eth.Deposit {
	panic("implement me")
}

func (BeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	panic("implement me")
}

func (BeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	panic("implement me")
}

func (BeaconBlockBody) IsNil() bool {
	return false
}

func (BeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	panic("implement me")
}

func (BeaconBlockBody) Proto() proto.Message {
	panic("implement me")
}

func (BeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	panic("implement me")
}

var _ block.SignedBeaconBlock = &SignedBeaconBlock{}
var _ block.BeaconBlock = &BeaconBlock{}
var _ block.BeaconBlockBody = &BeaconBlockBody{}
