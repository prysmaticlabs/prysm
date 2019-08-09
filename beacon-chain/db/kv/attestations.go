package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Attestation retrieval by root.
func (k *KVStore) Attestation(ctx context.Context, attRoot [32]byte) (*ethpb.Attestation, error) {
	return nil, nil
}

// Attestations retrieves a list of attestations by filter criteria.
func (k *KVStore) Attestations(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.Attestation, error) {
	return nil, nil
}

// HasAttestation checks if an attestation by root exists in the db.
func (k *KVStore) HasAttestation(ctx context.Context, attRoot [32]byte) bool {
	return false
}

// DeleteAttestation by root.
func (k *KVStore) DeleteAttestation(ctx context.Context, attRoot [32]byte) error {
	return nil
}

// SaveAttestation to the db.
func (k *KVStore) SaveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	return nil
}

// SaveAttestations via batch updates to the db.
func (k *KVStore) SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error {
	return nil
}
