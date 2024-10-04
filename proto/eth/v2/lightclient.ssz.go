package eth

import (
	"fmt"

	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
)

func (x *LightClientUpdate) ToInternal() (*LightClientUpdateInternal, error) {
	var attested ssz.Marshaler
	switch h := x.AttestedHeader.GetHeader().(type) {
	case *LightClientHeaderContainer_HeaderDeneb:
		attested = h.HeaderDeneb
	case *LightClientHeaderContainer_HeaderCapella:
		attested = h.HeaderCapella
	case *LightClientHeaderContainer_HeaderAltair:
		attested = h.HeaderAltair
	default:
		return nil, fmt.Errorf("unsupported header type %T", h)
	}
	attestedSSZ, err := attested.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal attested header into SSZ")
	}

	var finalized ssz.Marshaler
	switch h := x.FinalizedHeader.GetHeader().(type) {
	case *LightClientHeaderContainer_HeaderDeneb:
		finalized = h.HeaderDeneb
	case *LightClientHeaderContainer_HeaderCapella:
		finalized = h.HeaderCapella
	case *LightClientHeaderContainer_HeaderAltair:
		finalized = h.HeaderAltair
	default:
		return nil, fmt.Errorf("unsupported header type %T", h)
	}
	finalizedSSZ, err := finalized.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal finalized header into SSZ")
	}

	return &LightClientUpdateInternal{
		AttestedHeader:          attestedSSZ,
		NextSyncCommittee:       x.NextSyncCommittee,
		NextSyncCommitteeBranch: x.NextSyncCommitteeBranch,
		FinalizedHeader:         finalizedSSZ,
		FinalityBranch:          x.FinalityBranch,
		SyncAggregate:           x.SyncAggregate,
		SignatureSlot:           x.SignatureSlot,
	}, nil
}

func (x *LightClientUpdate) MarshalSSZ() ([]byte, error) {
	internal, err := x.ToInternal()
	if err != nil {
		return nil, errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.MarshalSSZ()
}

func (x *LightClientUpdate) MarshalSSZTo(buf []byte) ([]byte, error) {
	internal, err := x.ToInternal()
	if err != nil {
		return nil, errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.MarshalSSZTo(buf)
}

func (x *LightClientUpdate) UnmarshalSSZ(buf []byte) error {
	internal, err := x.ToInternal()
	if err != nil {
		return errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.UnmarshalSSZ(buf)
}

func (x *LightClientUpdate) SizeSSZ() int {
	internal, err := x.ToInternal()
	if err != nil {
		return -1
	}
	return internal.SizeSSZ()
}

func (x *LightClientFinalityUpdate) ToInternal() (*LightClientFinalityUpdateInternal, error) {
	var attested ssz.Marshaler
	switch h := x.AttestedHeader.GetHeader().(type) {
	case *LightClientHeaderContainer_HeaderDeneb:
		attested = h.HeaderDeneb
	case *LightClientHeaderContainer_HeaderCapella:
		attested = h.HeaderCapella
	case *LightClientHeaderContainer_HeaderAltair:
		attested = h.HeaderAltair
	default:
		return nil, fmt.Errorf("unsupported header type %T", h)
	}
	attestedSSZ, err := attested.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal attested header into SSZ")
	}

	var finalized ssz.Marshaler
	switch h := x.FinalizedHeader.GetHeader().(type) {
	case *LightClientHeaderContainer_HeaderDeneb:
		finalized = h.HeaderDeneb
	case *LightClientHeaderContainer_HeaderCapella:
		finalized = h.HeaderCapella
	case *LightClientHeaderContainer_HeaderAltair:
		finalized = h.HeaderAltair
	default:
		return nil, fmt.Errorf("unsupported header type %T", h)
	}
	finalizedSSZ, err := finalized.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal finalized header into SSZ")
	}

	return &LightClientFinalityUpdateInternal{
		AttestedHeader:  attestedSSZ,
		FinalizedHeader: finalizedSSZ,
		FinalityBranch:  x.FinalityBranch,
		SyncAggregate:   x.SyncAggregate,
		SignatureSlot:   x.SignatureSlot,
	}, nil
}

func (x *LightClientFinalityUpdate) MarshalSSZ() ([]byte, error) {
	internal, err := x.ToInternal()
	if err != nil {
		return nil, errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.MarshalSSZ()
}

func (x *LightClientFinalityUpdate) MarshalSSZTo(buf []byte) ([]byte, error) {
	internal, err := x.ToInternal()
	if err != nil {
		return nil, errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.MarshalSSZTo(buf)
}

func (x *LightClientFinalityUpdate) UnmarshalSSZ(buf []byte) error {
	internal, err := x.ToInternal()
	if err != nil {
		return errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.UnmarshalSSZ(buf)
}

func (x *LightClientFinalityUpdate) SizeSSZ() int {
	internal, err := x.ToInternal()
	if err != nil {
		return -1
	}
	return internal.SizeSSZ()
}

func (x *LightClientOptimisticUpdate) ToInternal() (*LightClientOptimisticUpdateInternal, error) {
	var attested ssz.Marshaler
	switch h := x.AttestedHeader.GetHeader().(type) {
	case *LightClientHeaderContainer_HeaderDeneb:
		attested = h.HeaderDeneb
	case *LightClientHeaderContainer_HeaderCapella:
		attested = h.HeaderCapella
	case *LightClientHeaderContainer_HeaderAltair:
		attested = h.HeaderAltair
	default:
		return nil, fmt.Errorf("unsupported header type %T", h)
	}
	attestedSSZ, err := attested.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal attested header into SSZ")
	}

	return &LightClientOptimisticUpdateInternal{
		AttestedHeader: attestedSSZ,
		SyncAggregate:  x.SyncAggregate,
		SignatureSlot:  x.SignatureSlot,
	}, nil
}

func (x *LightClientOptimisticUpdate) MarshalSSZ() ([]byte, error) {
	internal, err := x.ToInternal()
	if err != nil {
		return nil, errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.MarshalSSZ()
}

func (x *LightClientOptimisticUpdate) MarshalSSZTo(buf []byte) ([]byte, error) {
	internal, err := x.ToInternal()
	if err != nil {
		return nil, errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.MarshalSSZTo(buf)
}

func (x *LightClientOptimisticUpdate) UnmarshalSSZ(buf []byte) error {
	internal, err := x.ToInternal()
	if err != nil {
		return errors.Wrap(err, "could not convert update to internal representation")
	}
	return internal.UnmarshalSSZ(buf)
}

func (x *LightClientOptimisticUpdate) SizeSSZ() int {
	internal, err := x.ToInternal()
	if err != nil {
		return -1
	}
	return internal.SizeSSZ()
}
