package db

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	bolt "go.etcd.io/bbolt"

	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
)

func TestSplit(t *testing.T) {
	pubKeys := [][48]byte{{1}, {2}}
	sourceStore := SetupDB(t, pubKeys)

	proposalEpoch := uint64(0)
	proposalHistory1 := bitfield.Bitlist{0x01, 0x00, 0x00, 0x00, 0x01}
	if err := sourceStore.SaveProposalHistoryForEpoch(context.Background(), pubKeys[0][:], proposalEpoch, proposalHistory1); err != nil {
		t.Fatal("Saving proposal history failed")
	}
	proposalHistory2 := bitfield.Bitlist{0x02, 0x00, 0x00, 0x00, 0x01}
	if err := sourceStore.SaveProposalHistoryForEpoch(context.Background(), pubKeys[1][:], proposalEpoch, proposalHistory2); err != nil {
		t.Fatal("Saving proposal history failed")
	}

	attestationHistoryMap1 := make(map[uint64]uint64)
	attestationHistoryMap1[0] = 0
	pubKeyAttestationHistory1 := &slashpb.AttestationHistory{
		TargetToSource:     attestationHistoryMap1,
		LatestEpochWritten: 0,
	}
	attestationHistoryMap2 := make(map[uint64]uint64)
	attestationHistoryMap2[0] = 1
	pubKeyAttestationHistory2 := &slashpb.AttestationHistory{
		TargetToSource:     attestationHistoryMap2,
		LatestEpochWritten: 0,
	}
	dbAttestationHistory := make(map[[48]byte]*slashpb.AttestationHistory)
	dbAttestationHistory[pubKeys[0]] = pubKeyAttestationHistory1
	dbAttestationHistory[pubKeys[1]] = pubKeyAttestationHistory2
	if err := sourceStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory); err != nil {
		t.Fatalf("Saving attestation history failed %v", err)
	}

	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		if err := os.RemoveAll(targetDirectory); err != nil {
			t.Errorf("Could not remove target directory : %v", err)
		}
	})

	if err := Split(context.Background(), sourceStore, targetDirectory); err != nil {
		t.Fatalf("Splitting failed: %v", err)
	}

	encodedKey1 := hex.EncodeToString(pubKeys[0][:])[:12]
	keyStore1, err := GetKVStore(filepath.Join(targetDirectory, encodedKey1))
	if err != nil {
		t.Fatalf("Retrieving the merged store failed: %v", err)
	}
	if keyStore1 == nil {
		t.Fatalf("Retrieving target store for public key %v failed", encodedKey1)
	}
	if err := keyStore1.view(func(tx *bolt.Tx) error {
		otherKeyProposalsBucket := tx.Bucket(historicProposalsBucket).Bucket(pubKeys[1][:])
		if otherKeyProposalsBucket != nil {
			t.Errorf("Target store for public key %v contains proposals for another key", encodedKey1)
		}
		otherKeyAttestationsBucket := tx.Bucket(historicAttestationsBucket).Bucket(pubKeys[1][:])
		if otherKeyAttestationsBucket != nil {
			t.Errorf("Target store for public key %v contains attestations for another key", encodedKey1)
		}

		return nil
	}); err != nil {
		t.Fatal("Failed to close target store")
	}

	encodedKey2 := hex.EncodeToString(pubKeys[1][:])[:12]
	keyStore2, err := GetKVStore(filepath.Join(targetDirectory, encodedKey2))
	if err != nil {
		t.Fatalf("Retrieving the merged store failed: %v", err)
	}
	if keyStore2 == nil {
		t.Fatalf("Retrieving target store for public key %v failed", encodedKey2)
	}
	if err := keyStore2.view(func(tx *bolt.Tx) error {
		otherKeyProposalsBucket := tx.Bucket(historicProposalsBucket).Bucket(pubKeys[0][:])
		if otherKeyProposalsBucket != nil {
			t.Errorf("Target store for public key %v contains proposals for another key", encodedKey2)
		}
		otherKeyAttestationsBucket := tx.Bucket(historicAttestationsBucket).Bucket(pubKeys[0][:])
		if otherKeyAttestationsBucket != nil {
			t.Errorf("Target store for public key %v contains attestations for another key", encodedKey1)
		}

		return nil
	}); err != nil {
		t.Fatal("Failed to close target store")
	}

	splitProposalHistory1, err := keyStore1.ProposalHistoryForEpoch(
		context.Background(), pubKeys[0][:], proposalEpoch)
	if err != nil {
		t.Errorf("Retrieving split proposal history failed for public key %v", encodedKey1)
	} else {
		if !bytes.Equal(splitProposalHistory1, proposalHistory1) {
			t.Errorf(
				"Proposals not split correctly: expected %v vs received %v",
				proposalHistory1,
				splitProposalHistory1)
		}
	}
	splitProposalHistory2, err := keyStore2.ProposalHistoryForEpoch(
		context.Background(), pubKeys[1][:], proposalEpoch)
	if err != nil {
		t.Errorf("Retrieving split proposal history failed for public key %v", encodedKey2)
	} else {
		if !bytes.Equal(splitProposalHistory2, proposalHistory2) {
			t.Errorf(
				"Proposals not split correctly: expected %v vs received %v",
				proposalHistory2,
				splitProposalHistory2)
		}
	}

	splitAttestationsHistory1, err := keyStore1.AttestationHistoryForPubKeys(context.Background(), [][48]byte{pubKeys[0]})
	if err != nil {
		t.Errorf("Retrieving split attestation history failed for public key %v", encodedKey1)
	} else {
		if splitAttestationsHistory1[pubKeys[0]].TargetToSource[0] != attestationHistoryMap1[0] {
			t.Errorf(
				"Attestations not merged correctly: expected %v vs received %v",
				attestationHistoryMap1[0],
				splitAttestationsHistory1[pubKeys[0]].TargetToSource[0])
		}
	}
	splitAttestationsHistory2, err := keyStore2.AttestationHistoryForPubKeys(context.Background(), [][48]byte{pubKeys[1]})
	if err != nil {
		t.Errorf("Retrieving split attestation history failed for public key %v", encodedKey2)
	} else {
		if splitAttestationsHistory2[pubKeys[1]].TargetToSource[0] != attestationHistoryMap2[0] {
			t.Errorf(
				"Attestations not merged correctly: expected %v vs received %v",
				attestationHistoryMap2[0],
				splitAttestationsHistory2[pubKeys[1]].TargetToSource[0])
		}
	}
}
