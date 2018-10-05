package db

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// HasAttestation checks an attestation exists in beacon chain db by inputting its hash.
func (db *BeaconDB) HasAttestation(attestationHash [32]byte) (bool, error) {
	return db.has(attestationKey(attestationHash))
}

// SaveAttestation puts the attestation record into the beacon chain db.
func (db *BeaconDB) SaveAttestation(attestation *types.Attestation) error {
	hash := attestation.Key()
	key := attestationKey(hash)
	encodedState, err := attestation.Marshal()
	if err != nil {
		return err
	}
	return db.put(key, encodedState)
}

// GetAttestation retrieves an attestation record from the db using its hash.
func (db *BeaconDB) GetAttestation(hash [32]byte) (*types.Attestation, error) {
	key := attestationKey(hash)
	enc, err := db.get(key)
	if err != nil {
		return nil, err
	}

	attestation := &pb.AggregatedAttestation{}

	err = proto.Unmarshal(enc, attestation)

	return types.NewAttestation(attestation), err
}

// RemoveAttestation removes the attestation from the db.
func (db *BeaconDB) RemoveAttestation(blockHash [32]byte) error {
	return db.delete(attestationKey(blockHash))
}

// HasAttestationHash checks if the beacon block has the attestation.
func (db *BeaconDB) HasAttestationHash(blockHash [32]byte, attestationHash [32]byte) (bool, error) {
	enc, err := db.get(attestationHashListKey(blockHash))
	if err != nil {
		return false, err
	}

	attestationHashes := &pb.AttestationHashes{}
	if err := proto.Unmarshal(enc, attestationHashes); err != nil {
		return false, err
	}

	for _, hash := range attestationHashes.AttestationHash {
		if bytes.Equal(hash, attestationHash[:]) {
			return true, nil
		}
	}
	return false, nil
}

// HasAttestationHashList checks if the attestation hash list is available.
func (db *BeaconDB) HasAttestationHashList(blockHash [32]byte) (bool, error) {
	key := attestationHashListKey(blockHash)

	hasKey, err := db.has(key)
	if err != nil {
		return false, err
	}
	return hasKey, nil
}

// GetAttestationHashList gets the attestation hash list of the beacon block from the db.
func (db *BeaconDB) GetAttestationHashList(blockHash [32]byte) ([][]byte, error) {
	key := attestationHashListKey(blockHash)

	hasList, err := db.HasAttestationHashList(blockHash)
	if err != nil {
		return [][]byte{}, err
	}
	if !hasList {
		if err := db.put(key, []byte{}); err != nil {
			return [][]byte{}, err
		}
	}
	enc, err := db.get(key)
	if err != nil {
		return [][]byte{}, err
	}

	attestationHashes := &pb.AttestationHashes{}
	if err := proto.Unmarshal(enc, attestationHashes); err != nil {
		return [][]byte{}, err
	}
	return attestationHashes.AttestationHash, nil
}

// RemoveAttestationHashList removes the attestation hash list of the beacon block from the db.
func (db *BeaconDB) RemoveAttestationHashList(blockHash [32]byte) error {
	return db.delete(attestationHashListKey(blockHash))
}

// SaveAttestationHash saves the attestation hash into the attestation hash list of the corresponding beacon block.
func (db *BeaconDB) SaveAttestationHash(blockHash [32]byte, attestationHash [32]byte) error {
	key := attestationHashListKey(blockHash)

	hashes, err := db.GetAttestationHashList(blockHash)
	if err != nil {
		return err
	}
	hashes = append(hashes, attestationHash[:])

	attestationHashes := &pb.AttestationHashes{}
	attestationHashes.AttestationHash = hashes

	encodedState, err := proto.Marshal(attestationHashes)
	if err != nil {
		return err
	}

	return db.put(key, encodedState)
}
