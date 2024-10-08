package light_client

import (
	"fmt"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	consensustypes "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"google.golang.org/protobuf/proto"
)

func NewWrappedBootstrap(m proto.Message) (interfaces.LightClientBootstrap, error) {
	if m == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	switch t := m.(type) {
	case *pb.LightClientBootstrapAltair:
		return NewWrappedBootstrapAltair(t)
	case *pb.LightClientBootstrapCapella:
		return NewWrappedBootstrapCapella(t)
	case *pb.LightClientBootstrapDeneb:
		return NewWrappedBootstrapDeneb(t)
	default:
		return nil, fmt.Errorf("cannot construct light client bootstrap from type %T", t)
	}
}

type bootstrapAltair struct {
	p                          *pb.LightClientBootstrapAltair
	header                     interfaces.LightClientHeader
	currentSyncCommitteeBranch interfaces.LightClientSyncCommitteeBranch
}

var _ interfaces.LightClientBootstrap = &bootstrapAltair{}

func NewWrappedBootstrapAltair(p *pb.LightClientBootstrapAltair) (interfaces.LightClientBootstrap, error) {
	if p == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	header, err := NewWrappedHeaderAltair(p.Header)
	if err != nil {
		return nil, err
	}
	branch, err := createBranch[interfaces.LightClientSyncCommitteeBranch](
		"sync committee",
		p.CurrentSyncCommitteeBranch,
		fieldparams.SyncCommitteeBranchDepth,
	)
	if err != nil {
		return nil, err
	}

	return &bootstrapAltair{
		p:                          p,
		header:                     header,
		currentSyncCommitteeBranch: branch,
	}, nil
}

func (h *bootstrapAltair) MarshalSSZTo(dst []byte) ([]byte, error) {
	return h.p.MarshalSSZTo(dst)
}

func (h *bootstrapAltair) MarshalSSZ() ([]byte, error) {
	return h.p.MarshalSSZ()
}

func (h *bootstrapAltair) SizeSSZ() int {
	return h.p.SizeSSZ()
}

func (h *bootstrapAltair) Version() int {
	return version.Altair
}

func (h *bootstrapAltair) Header() interfaces.LightClientHeader {
	return h.header
}

func (h *bootstrapAltair) CurrentSyncCommittee() *pb.SyncCommittee {
	return h.p.CurrentSyncCommittee
}

func (h *bootstrapAltair) CurrentSyncCommitteeBranch() interfaces.LightClientSyncCommitteeBranch {
	return h.currentSyncCommitteeBranch
}

type bootstrapCapella struct {
	p                          *pb.LightClientBootstrapCapella
	header                     interfaces.LightClientHeader
	currentSyncCommitteeBranch interfaces.LightClientSyncCommitteeBranch
}

var _ interfaces.LightClientBootstrap = &bootstrapCapella{}

func NewWrappedBootstrapCapella(p *pb.LightClientBootstrapCapella) (interfaces.LightClientBootstrap, error) {
	if p == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	header, err := NewWrappedHeaderCapella(p.Header)
	if err != nil {
		return nil, err
	}
	branch, err := createBranch[interfaces.LightClientSyncCommitteeBranch](
		"sync committee",
		p.CurrentSyncCommitteeBranch,
		fieldparams.SyncCommitteeBranchDepth,
	)
	if err != nil {
		return nil, err
	}

	return &bootstrapCapella{
		p:                          p,
		header:                     header,
		currentSyncCommitteeBranch: branch,
	}, nil
}

func (h *bootstrapCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	return h.p.MarshalSSZTo(dst)
}

func (h *bootstrapCapella) MarshalSSZ() ([]byte, error) {
	return h.p.MarshalSSZ()
}

func (h *bootstrapCapella) SizeSSZ() int {
	return h.p.SizeSSZ()
}

func (h *bootstrapCapella) Version() int {
	return version.Capella
}

func (h *bootstrapCapella) Header() interfaces.LightClientHeader {
	return h.header
}

func (h *bootstrapCapella) CurrentSyncCommittee() *pb.SyncCommittee {
	return h.p.CurrentSyncCommittee
}

func (h *bootstrapCapella) CurrentSyncCommitteeBranch() interfaces.LightClientSyncCommitteeBranch {
	return h.currentSyncCommitteeBranch
}

type bootstrapDeneb struct {
	p                          *pb.LightClientBootstrapDeneb
	header                     interfaces.LightClientHeader
	currentSyncCommitteeBranch interfaces.LightClientSyncCommitteeBranch
}

var _ interfaces.LightClientBootstrap = &bootstrapDeneb{}

func NewWrappedBootstrapDeneb(p *pb.LightClientBootstrapDeneb) (interfaces.LightClientBootstrap, error) {
	if p == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	header, err := NewWrappedHeaderDeneb(p.Header)
	if err != nil {
		return nil, err
	}
	branch, err := createBranch[interfaces.LightClientSyncCommitteeBranch](
		"sync committee",
		p.CurrentSyncCommitteeBranch,
		fieldparams.SyncCommitteeBranchDepth,
	)
	if err != nil {
		return nil, err
	}

	return &bootstrapDeneb{
		p:                          p,
		header:                     header,
		currentSyncCommitteeBranch: branch,
	}, nil
}

func (h *bootstrapDeneb) MarshalSSZTo(dst []byte) ([]byte, error) {
	return h.p.MarshalSSZTo(dst)
}

func (h *bootstrapDeneb) MarshalSSZ() ([]byte, error) {
	return h.p.MarshalSSZ()
}

func (h *bootstrapDeneb) SizeSSZ() int {
	return h.p.SizeSSZ()
}

func (h *bootstrapDeneb) Version() int {
	return version.Deneb
}

func (h *bootstrapDeneb) Header() interfaces.LightClientHeader {
	return h.header
}

func (h *bootstrapDeneb) CurrentSyncCommittee() *pb.SyncCommittee {
	return h.p.CurrentSyncCommittee
}

func (h *bootstrapDeneb) CurrentSyncCommitteeBranch() interfaces.LightClientSyncCommitteeBranch {
	return h.currentSyncCommitteeBranch
}
