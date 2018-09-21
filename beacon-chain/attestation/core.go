package attestation

import (
	"bytes"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Handler represents the core attestation object
// containing a db.
type Handler struct {
	db ethdb.Database
}

// NewHandler initializes an attestation handler.
func NewHandler(db ethdb.Database) (*Handler, error) {
	handler := &Handler{
		db: db,
	}

	return handler, nil
}

func (h *Handler) hasAttestation(attestationHash [32]byte) (bool, error) {
	return h.db.Has(blockchain.AttestationKey(attestationHash))
}

// saveAttestation puts the attestation record into the beacon chain db.
func (h *Handler) saveAttestation(attestation *types.Attestation) error {
	hash := attestation.Key()
	key := blockchain.AttestationKey(hash)
	encodedState, err := attestation.Marshal()
	if err != nil {
		return err
	}
	return h.db.Put(key, encodedState)
}

// getAttestation retrieves an attestation record from the db using its hash.
func (h *Handler) getAttestation(hash [32]byte) (*types.Attestation, error) {
	key := blockchain.AttestationKey(hash)
	enc, err := h.db.Get(key)
	if err != nil {
		return nil, err
	}

	attestation := &pb.AggregatedAttestation{}

	err = proto.Unmarshal(enc, attestation)

	return types.NewAttestation(attestation), err
}

// removeAttestation removes the attestation from the db.
func (h *Handler) removeAttestation(blockHash [32]byte) error {
	return h.db.Delete(blockchain.AttestationKey(blockHash))
}

// hasAttestationHash checks if the beacon block has the attestation.
func (h *Handler) hasAttestationHash(blockHash [32]byte, attestationHash [32]byte) (bool, error) {
	enc, err := h.db.Get(blockchain.AttestationHashListKey(blockHash))
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

// hasAttestationHashList checks if the attestation hash list is available.
func (h *Handler) hasAttestationHashList(blockHash [32]byte) (bool, error) {
	key := blockchain.AttestationHashListKey(blockHash)

	hasKey, err := h.db.Has(key)
	if err != nil {
		return false, err
	}
	if !hasKey {
		return false, nil
	}
	return true, nil
}

// getAttestationHashList gets the attestation hash list of the beacon block from the db.
func (h *Handler) getAttestationHashList(blockHash [32]byte) ([][]byte, error) {
	key := blockchain.AttestationHashListKey(blockHash)

	hasList, err := h.hasAttestationHashList(blockHash)
	if err != nil {
		return [][]byte{}, err
	}
	if !hasList {
		if err := h.db.Put(key, []byte{}); err != nil {
			return [][]byte{}, err
		}
	}
	enc, err := h.db.Get(key)
	if err != nil {
		return [][]byte{}, err
	}

	attestationHashes := &pb.AttestationHashes{}
	if err := proto.Unmarshal(enc, attestationHashes); err != nil {
		return [][]byte{}, err
	}
	return attestationHashes.AttestationHash, nil
}

// removeAttestationHashList removes the attestation hash list of the beacon block from the db.
func (h *Handler) removeAttestationHashList(blockHash [32]byte) error {
	return h.db.Delete(blockchain.AttestationHashListKey(blockHash))
}

// saveAttestationHash saves the attestation hash into the attestation hash list of the corresponding beacon block.
func (h *Handler) saveAttestationHash(blockHash [32]byte, attestationHash [32]byte) error {
	key := blockchain.AttestationHashListKey(blockHash)

	hashes, err := h.getAttestationHashList(blockHash)
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

	return h.db.Put(key, encodedState)
}
