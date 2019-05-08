package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func createAttestation(enc []byte) (*pbp2p.Attestation, error) {
	protoAttestation := &pbp2p.Attestation{}
	err := proto.Unmarshal(enc, protoAttestation)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoAttestation, nil
}

// SaveAttestation TODO
func (db *ValidatorDB) SaveAttestation(fork *pbp2p.Fork, pubKey *bls.PublicKey, attestation *pbp2p.Attestation) error {
	epoch := attestation.Data.Slot / params.BeaconConfig().SlotsPerEpoch
	if lastAttestationEpoch, ok := db.lastAttestationEpoch[(*pubKey)]; !ok || lastAttestationEpoch < epoch {
		db.lastAttestationEpoch[(*pubKey)] = epoch
	}

	forkVersion := forkutil.ForkVersion(fork, epoch)
	attestationEnc, err := attestation.Marshal()
	if err != nil {
		return fmt.Errorf("failed to encode attestation: %v", err)
	}

	return db.update(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, pubKey, forkVersion, attestationBucket, true)
		return bucket.Put(bytesutil.Bytes8(epoch), attestationEnc)
	})
}

// GetAttestation
func (db *ValidatorDB) GetAttestation(fork *pbp2p.Fork, pubKey *bls.PublicKey, epoch uint64) (attestation *pbp2p.Attestation, err error) {
	if lastAttestationEpoch, ok := db.lastAttestationEpoch[(*pubKey)]; ok && lastAttestationEpoch < epoch {
		fmt.Printf("did not perform the attsation for the epoch or higher\n")
		return
	}

	forkVersion := forkutil.ForkVersion(fork, epoch)
	err = db.view(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, pubKey, forkVersion, attestationBucket, false)
		if bucket == nil {
			return nil
		}
		attestationEnc := bucket.Get(bytesutil.Bytes8(epoch))
		attestation, err = createAttestation(attestationEnc)
		return err
	})
	return
}

func (db *ValidatorDB) getMaxAttestationEpoch(pubKey *bls.PublicKey) (maxAttestationEpoch uint64, err error) {
	err = db.lastInAllForks(pubKey, attestationBucket, func(_, lastInForkEnc []byte) error {
		lastInFork, err := createAttestation(lastInForkEnc)
		if err != nil {
			log.WithError(err).Error("can't create attestation")
			return err
		}

		maxAttestationEpoch = lastInFork.Data.Slot / params.BeaconConfig().SlotsPerEpoch
		return nil
	})
	return
}
