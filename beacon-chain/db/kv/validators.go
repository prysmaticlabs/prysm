package kv

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Attestation retrieval by root.
func (k *Store) ValidatorLatestVote(ctx context.Context, validatorIdx uint64) (*pb.ValidatorLatestVote, error) {
	return nil, nil
}

// Attestations retrieves a list of attestations by filter criteria.
func (k *Store) HasValidatorLatestVote(ctx context.Context, validatorIdx uint64) bool {
	return false
}

// HasAttestation checks if an attestation by root exists in the db.
func (k *Store) SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *pb.ValidatorLatestVote) error {
	return nil
}

// DeleteAttestation by root.
func (k *Store) ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, error) {
	return 0, nil
}

// SaveAttestation to the db.
func (k *Store) HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool {
	return false
}

// SaveAttestations via batch updates to the db.
func (k *Store) DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error {
	return nil
}

// SaveAttestations via batch updates to the db.
func (k *Store) SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error {
	return nil
}
