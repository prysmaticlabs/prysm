package kv

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	bolt "go.etcd.io/bbolt"
)

type storeHistory struct {
	Proposals    map[[48]byte]bitfield.Bitlist
	Attestations map[[48]byte]map[uint64]uint64
}

func TestMerge(t *testing.T) {
	firstStorePubKeys := [][48]byte{{1}, {2}}
	firstStore := setupDB(t, firstStorePubKeys)
	secondStorePubKeys := [][48]byte{{3}, {4}}
	secondStore := setupDB(t, secondStorePubKeys)

	storeHistory1, err := prepareStore(firstStore, firstStorePubKeys)
	if err != nil {
		t.Fatal(err)
	}
	storeHistory2, err := prepareStore(secondStore, secondStorePubKeys)
	if err != nil {
		t.Fatal(err)
	}
	mergedProposals := make(map[[48]byte]bitfield.Bitlist)
	for k, v := range storeHistory1.Proposals {
		mergedProposals[k] = v
	}
	for k, v := range storeHistory2.Proposals {
		mergedProposals[k] = v
	}
	mergedAttestations := make(map[[48]byte]map[uint64]uint64)
	for k, v := range storeHistory1.Attestations {
		mergedAttestations[k] = v
	}
	for k, v := range storeHistory2.Attestations {
		mergedAttestations[k] = v
	}
	mergedStoreHistory := storeHistory{
		Proposals:    mergedProposals,
		Attestations: mergedAttestations,
	}

	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		if err := os.RemoveAll(targetDirectory); err != nil {
			t.Errorf("Could not remove target directory : %v", err)
		}
	})

	err = Merge(context.Background(), []*Store{firstStore, secondStore}, targetDirectory)
	if err != nil {
		t.Fatalf("Merging failed: %v", err)
	}
	mergedStore, err := GetKVStore(targetDirectory)
	if err != nil {
		t.Fatalf("Retrieving the merged store failed: %v", err)
	}

	assertStore(
		t,
		mergedStore,
		append(firstStorePubKeys, secondStorePubKeys[0], secondStorePubKeys[1]),
		&mergedStoreHistory)
}

func TestSplit(t *testing.T) {
	pubKey1 := [48]byte{1}
	pubKey2 := [48]byte{2}
	sourceStore := setupDB(t, [][48]byte{pubKey1, pubKey2})

	storeHistory1, err := prepareStore(sourceStore, [][48]byte{pubKey1})
	if err != nil {
		t.Fatal(err)
	}
	storeHistory2, err := prepareStore(sourceStore, [][48]byte{pubKey2})
	if err != nil {
		t.Fatal(err)
	}

	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		if err := os.RemoveAll(targetDirectory); err != nil {
			t.Errorf("Could not remove target directory: %v", err)
		}
	})

	if err := Split(context.Background(), sourceStore, targetDirectory); err != nil {
		t.Fatalf("Splitting failed: %v", err)
	}

	encodedKey1 := hex.EncodeToString(pubKey1[:])[:12]
	keyStore1, err := GetKVStore(filepath.Join(targetDirectory, encodedKey1))
	if err != nil {
		t.Fatalf("Retrieving the store for public key %v failed: %v", encodedKey1, err)
	}
	if keyStore1 == nil {
		t.Fatalf("No store created for public key %v", encodedKey1)
	}

	encodedKey2 := hex.EncodeToString(pubKey2[:])[:12]
	keyStore2, err := GetKVStore(filepath.Join(targetDirectory, encodedKey2))
	if err != nil {
		t.Fatalf("Retrieving the store for public key %v failed: %v", encodedKey2, err)
	}
	if keyStore2 == nil {
		t.Fatalf("No store created for public key %v", encodedKey2)
	}

	if err := keyStore1.view(func(tx *bolt.Tx) error {
		otherKeyProposalsBucket := tx.Bucket(historicProposalsBucket).Bucket(pubKey2[:])
		if otherKeyProposalsBucket != nil {
			t.Fatalf("Store for public key %v contains proposals for another key", encodedKey2)
		}
		otherKeyAttestationsBucket := tx.Bucket(historicAttestationsBucket).Bucket(pubKey2[:])
		if otherKeyAttestationsBucket != nil {
			t.Fatalf("Store for public key %v contains attestations for another key", encodedKey2)
		}

		return nil
	}); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}
	if err := keyStore2.view(func(tx *bolt.Tx) error {
		otherKeyProposalsBucket := tx.Bucket(historicProposalsBucket).Bucket(pubKey1[:])
		if otherKeyProposalsBucket != nil {
			t.Fatalf("Store for public key %v contains proposals for another key", encodedKey1)
		}
		otherKeyAttestationsBucket := tx.Bucket(historicAttestationsBucket).Bucket(pubKey1[:])
		if otherKeyAttestationsBucket != nil {
			t.Fatalf("Store for public key %v contains attestations for another key", encodedKey1)
		}

		return nil
	}); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	assertStore(t, keyStore1, [][48]byte{pubKey1}, storeHistory1)
	assertStore(t, keyStore2, [][48]byte{pubKey2}, storeHistory2)
}

func TestSplit_AttestationsWithoutMatchingProposalsAreSplit(t *testing.T) {
	pubKey1 := [48]byte{1}
	pubKey2 := [48]byte{2}
	sourceStore := setupDB(t, [][48]byte{pubKey1, pubKey2})

	_, err := prepareStoreProposals(sourceStore, [][48]byte{pubKey1})
	if err != nil {
		t.Fatal(err)
	}
	attestationHistory, err := prepareStoreAttestations(sourceStore, [][48]byte{pubKey1, pubKey2})
	if err != nil {
		t.Fatal(err)
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

	encodedKey1 := hex.EncodeToString(pubKey1[:])[:12]
	encodedKey2 := hex.EncodeToString(pubKey2[:])[:12]

	attestationsOnlyKeyStore, err := GetKVStore(filepath.Join(targetDirectory, encodedKey2))
	if err != nil {
		t.Fatalf("Retrieving the store failed: %v", err)
	}
	if attestationsOnlyKeyStore == nil {
		t.Fatalf("No store created for public key %v", encodedKey2)
	}
	if err := attestationsOnlyKeyStore.view(func(tx *bolt.Tx) error {
		otherKeyProposalsBucket := tx.Bucket(historicProposalsBucket).Bucket(pubKey1[:])
		if otherKeyProposalsBucket != nil {
			t.Fatalf("Store for public key %v contains proposals for another key", encodedKey1)
		}
		otherKeyAttestationsBucket := tx.Bucket(historicAttestationsBucket).Bucket(pubKey1[:])
		if otherKeyAttestationsBucket != nil {
			t.Fatalf("Store for public key %v contains attestations for another key", encodedKey1)
		}

		return nil
	}); err != nil {
		t.Fatalf("Failed to retrieve attestations: %v", err)
	}

	splitAttestationsHistory, err :=
		attestationsOnlyKeyStore.AttestationHistoryForPubKeys(context.Background(), [][48]byte{pubKey2})
	if err != nil {
		t.Fatalf("Retrieving attestation history failed for public key %v", encodedKey2)
	}
	if splitAttestationsHistory[pubKey2].TargetToSource[0] != attestationHistory[pubKey2][0] {
		t.Fatalf(
			"Attestations not merged correctly: expected %v vs received %v",
			attestationHistory[pubKey2][0],
			splitAttestationsHistory[pubKey2].TargetToSource[0])
	}
}

func prepareStore(store *Store, pubKeys [][48]byte) (*storeHistory, error) {
	proposals, err := prepareStoreProposals(store, pubKeys)
	if err != nil {
		return nil, err
	}
	attestations, err := prepareStoreAttestations(store, pubKeys)
	if err != nil {
		return nil, err
	}
	history := storeHistory{
		Proposals:    proposals,
		Attestations: attestations,
	}
	return &history, nil
}

func prepareStoreProposals(store *Store, pubKeys [][48]byte) (map[[48]byte]bitfield.Bitlist, error) {
	proposals := make(map[[48]byte]bitfield.Bitlist)

	for i, key := range pubKeys {
		proposalHistory := bitfield.Bitlist{byte(i), 0x00, 0x00, 0x00, 0x01}
		if err := store.SaveProposalHistoryForEpoch(context.Background(), key[:], 0, proposalHistory); err != nil {
			return nil, errors.Wrapf(err, "Saving proposal history failed")
		}
		proposals[key] = proposalHistory
	}

	return proposals, nil
}

func prepareStoreAttestations(store *Store, pubKeys [][48]byte) (map[[48]byte]map[uint64]uint64, error) {
	storeAttestationHistory := make(map[[48]byte]*slashpb.AttestationHistory)
	attestations := make(map[[48]byte]map[uint64]uint64)

	for i, key := range pubKeys {
		attestationHistoryMap := make(map[uint64]uint64)
		attestationHistoryMap[0] = uint64(i)
		attestationHistory := &slashpb.AttestationHistory{
			TargetToSource:     attestationHistoryMap,
			LatestEpochWritten: 0,
		}
		storeAttestationHistory[key] = attestationHistory
		attestations[key] = attestationHistoryMap
	}
	if err := store.SaveAttestationHistoryForPubKeys(context.Background(), storeAttestationHistory); err != nil {
		return nil, errors.Wrapf(err, "Saving attestation history failed")
	}

	return attestations, nil
}

func assertStore(t *testing.T, store *Store, pubKeys [][48]byte, expectedHistory *storeHistory) {
	for _, key := range pubKeys {
		proposalHistory, err := store.ProposalHistoryForEpoch(
			context.Background(), key[:], 0)
		if err != nil {
			t.Fatalf("Retrieving proposal history failed for public key %v", key)
		}
		expectedProposals := expectedHistory.Proposals[key]
		if !bytes.Equal(proposalHistory, expectedProposals) {
			t.Fatalf("Proposals are incorrect: expected %v vs received %v", expectedProposals, proposalHistory)
		}
	}

	attestationHistory, err := store.AttestationHistoryForPubKeys(context.Background(), pubKeys)
	if err != nil {
		t.Fatalf("Retrieving attestation history failed")
	}
	for _, key := range pubKeys {
		expectedAttestations := expectedHistory.Attestations[key]
		if attestationHistory[key].TargetToSource[0] != expectedAttestations[0] {
			t.Fatalf(
				"Attestations are incorrect: expected %v vs received %v",
				expectedAttestations[0],
				attestationHistory[key].TargetToSource[0])
		}
	}
}
