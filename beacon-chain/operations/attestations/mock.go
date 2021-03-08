package attestations

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

type PoolMock struct {
	AggregatedAtts []*ethpb.Attestation
}

func (*PoolMock) AggregateUnaggregatedAttestations(_ context.Context) error {
	panic("implement me")
}

func (*PoolMock) AggregateUnaggregatedAttestationsBySlotIndex(_ context.Context, _ types.Slot, _ types.CommitteeIndex) error {
	panic("implement me")
}

func (*PoolMock) SaveAggregatedAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (m *PoolMock) SaveAggregatedAttestations(atts []*ethpb.Attestation) error {
	m.AggregatedAtts = append(m.AggregatedAtts, atts...)
	return nil
}

func (m *PoolMock) AggregatedAttestations() []*ethpb.Attestation {
	return m.AggregatedAtts
}

func (*PoolMock) AggregatedAttestationsBySlotIndex(_ context.Context, _ types.Slot, _ types.CommitteeIndex) []*ethpb.Attestation {
	panic("implement me")
}

func (*PoolMock) DeleteAggregatedAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) HasAggregatedAttestation(_ *ethpb.Attestation) (bool, error) {
	panic("implement me")
}

func (*PoolMock) AggregatedAttestationCount() int {
	panic("implement me")
}

func (*PoolMock) SaveUnaggregatedAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) SaveUnaggregatedAttestations(_ []*ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) UnaggregatedAttestations() ([]*ethpb.Attestation, error) {
	panic("implement me")
}

func (*PoolMock) UnaggregatedAttestationsBySlotIndex(_ context.Context, _ types.Slot, _ types.CommitteeIndex) []*ethpb.Attestation {
	panic("implement me")
}

func (*PoolMock) DeleteUnaggregatedAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) DeleteSeenUnaggregatedAttestations() (int, error) {
	panic("implement me")
}

func (*PoolMock) UnaggregatedAttestationCount() int {
	panic("implement me")
}

func (*PoolMock) SaveBlockAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) SaveBlockAttestations(_ []*ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) BlockAttestations() []*ethpb.Attestation {
	panic("implement me")
}

func (*PoolMock) DeleteBlockAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) SaveForkchoiceAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) SaveForkchoiceAttestations(_ []*ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) ForkchoiceAttestations() []*ethpb.Attestation {
	panic("implement me")
}

func (*PoolMock) DeleteForkchoiceAttestation(_ *ethpb.Attestation) error {
	panic("implement me")
}

func (*PoolMock) ForkchoiceAttestationCount() int {
	panic("implement me")
}
