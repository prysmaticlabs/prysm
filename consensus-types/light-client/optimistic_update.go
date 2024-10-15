package light_client

import (
	"fmt"

	consensustypes "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"google.golang.org/protobuf/proto"
)

func NewWrappedOptimisticUpdate(m proto.Message) (interfaces.LightClientOptimisticUpdate, error) {
	if m == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	switch t := m.(type) {
	case *pb.LightClientOptimisticUpdateAltair:
		return NewWrappedOptimisticUpdateAltair(t)
	case *pb.LightClientOptimisticUpdateCapella:
		return NewWrappedOptimisticUpdateCapella(t)
	case *pb.LightClientOptimisticUpdateDeneb:
		return NewWrappedOptimisticUpdateDeneb(t)
	default:
		return nil, fmt.Errorf("cannot construct light client optimistic update from type %T", t)
	}
}

type OptimisticUpdateAltair struct {
	p              *pb.LightClientOptimisticUpdateAltair
	attestedHeader interfaces.LightClientHeader
}

var _ interfaces.LightClientOptimisticUpdate = &OptimisticUpdateAltair{}

func NewWrappedOptimisticUpdateAltair(p *pb.LightClientOptimisticUpdateAltair) (interfaces.LightClientOptimisticUpdate, error) {
	if p == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	attestedHeader, err := NewWrappedHeaderAltair(p.AttestedHeader)
	if err != nil {
		return nil, err
	}

	return &OptimisticUpdateAltair{
		p:              p,
		attestedHeader: attestedHeader,
	}, nil
}

func (u *OptimisticUpdateAltair) MarshalSSZTo(dst []byte) ([]byte, error) {
	return u.p.MarshalSSZTo(dst)
}

func (u *OptimisticUpdateAltair) MarshalSSZ() ([]byte, error) {
	return u.p.MarshalSSZ()
}

func (u *OptimisticUpdateAltair) SizeSSZ() int {
	return u.p.SizeSSZ()
}

func (u *OptimisticUpdateAltair) Version() int {
	return version.Altair
}

func (u *OptimisticUpdateAltair) AttestedHeader() interfaces.LightClientHeader {
	return u.attestedHeader
}

func (u *OptimisticUpdateAltair) SyncAggregate() *pb.SyncAggregate {
	return u.p.SyncAggregate
}

func (u *OptimisticUpdateAltair) SignatureSlot() primitives.Slot {
	return u.p.SignatureSlot
}

type OptimisticUpdateCapella struct {
	p              *pb.LightClientOptimisticUpdateCapella
	attestedHeader interfaces.LightClientHeader
}

var _ interfaces.LightClientOptimisticUpdate = &OptimisticUpdateCapella{}

func NewWrappedOptimisticUpdateCapella(p *pb.LightClientOptimisticUpdateCapella) (interfaces.LightClientOptimisticUpdate, error) {
	if p == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	attestedHeader, err := NewWrappedHeaderCapella(p.AttestedHeader)
	if err != nil {
		return nil, err
	}

	return &OptimisticUpdateCapella{
		p:              p,
		attestedHeader: attestedHeader,
	}, nil
}

func (u *OptimisticUpdateCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	return u.p.MarshalSSZTo(dst)
}

func (u *OptimisticUpdateCapella) MarshalSSZ() ([]byte, error) {
	return u.p.MarshalSSZ()
}

func (u *OptimisticUpdateCapella) SizeSSZ() int {
	return u.p.SizeSSZ()
}

func (u *OptimisticUpdateCapella) Version() int {
	return version.Capella
}

func (u *OptimisticUpdateCapella) AttestedHeader() interfaces.LightClientHeader {
	return u.attestedHeader
}

func (u *OptimisticUpdateCapella) SyncAggregate() *pb.SyncAggregate {
	return u.p.SyncAggregate
}

func (u *OptimisticUpdateCapella) SignatureSlot() primitives.Slot {
	return u.p.SignatureSlot
}

type OptimisticUpdateDeneb struct {
	p              *pb.LightClientOptimisticUpdateDeneb
	attestedHeader interfaces.LightClientHeader
}

var _ interfaces.LightClientOptimisticUpdate = &OptimisticUpdateDeneb{}

func NewWrappedOptimisticUpdateDeneb(p *pb.LightClientOptimisticUpdateDeneb) (interfaces.LightClientOptimisticUpdate, error) {
	if p == nil {
		return nil, consensustypes.ErrNilObjectWrapped
	}
	attestedHeader, err := NewWrappedHeaderDeneb(p.AttestedHeader)
	if err != nil {
		return nil, err
	}

	return &OptimisticUpdateDeneb{
		p:              p,
		attestedHeader: attestedHeader,
	}, nil
}

func (u *OptimisticUpdateDeneb) MarshalSSZTo(dst []byte) ([]byte, error) {
	return u.p.MarshalSSZTo(dst)
}

func (u *OptimisticUpdateDeneb) MarshalSSZ() ([]byte, error) {
	return u.p.MarshalSSZ()
}

func (u *OptimisticUpdateDeneb) SizeSSZ() int {
	return u.p.SizeSSZ()
}

func (u *OptimisticUpdateDeneb) Version() int {
	return version.Deneb
}

func (u *OptimisticUpdateDeneb) AttestedHeader() interfaces.LightClientHeader {
	return u.attestedHeader
}

func (u *OptimisticUpdateDeneb) SyncAggregate() *pb.SyncAggregate {
	return u.p.SyncAggregate
}

func (u *OptimisticUpdateDeneb) SignatureSlot() primitives.Slot {
	return u.p.SignatureSlot
}
