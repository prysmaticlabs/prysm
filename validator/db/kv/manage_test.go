package kv

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, err)
	storeHistory2, err := prepareStore(secondStore, secondStorePubKeys)
	require.NoError(t, err)
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
		assert.NoError(t, os.RemoveAll(targetDirectory), "Could not remove target directory")
	})

	err = Merge(context.Background(), []*Store{firstStore, secondStore}, targetDirectory)
	require.NoError(t, err, "Merging failed")
	mergedStore, err := GetKVStore(targetDirectory)
	require.NoError(t, err, "Retrieving the merged store failed")

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
	require.NoError(t, err)
	storeHistory2, err := prepareStore(sourceStore, [][48]byte{pubKey2})
	require.NoError(t, err)

	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(targetDirectory), "Could not remove target directory")
	})

	require.NoError(t, Split(context.Background(), sourceStore, targetDirectory), "Splitting failed")

	encodedKey1 := hex.EncodeToString(pubKey1[:])[:12]
	keyStore1, err := GetKVStore(filepath.Join(targetDirectory, encodedKey1))
	require.NoError(t, err, "Retrieving the store for public key %v failed", encodedKey1)
	require.NotNil(t, keyStore1, "No store created for public key %v", encodedKey1)

	encodedKey2 := hex.EncodeToString(pubKey2[:])[:12]
	keyStore2, err := GetKVStore(filepath.Join(targetDirectory, encodedKey2))
	require.NoError(t, err, "Retrieving the store for public key %v failed", encodedKey2)
	require.NotNil(t, keyStore2, "No store created for public key %v", encodedKey2)

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
	require.NoError(t, err)
	attestationHistory, err := prepareStoreAttestations(sourceStore, [][48]byte{pubKey1, pubKey2})
	require.NoError(t, err)

	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(targetDirectory), "Could not remove target directory")
	})

	require.NoError(t, Split(context.Background(), sourceStore, targetDirectory), "Splitting failed")

	encodedKey1 := hex.EncodeToString(pubKey1[:])[:12]
	encodedKey2 := hex.EncodeToString(pubKey2[:])[:12]

	attestationsOnlyKeyStore, err := GetKVStore(filepath.Join(targetDirectory, encodedKey2))
	require.NoError(t, err, "Retrieving the store failed")
	require.NotNil(t, attestationsOnlyKeyStore, "No store created for public key %v", encodedKey2)

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
	require.NoError(t, err, "Retrieving attestation history failed for public key %v", encodedKey2)
	require.Equal(t, attestationHistory[pubKey2][0], splitAttestationsHistory[pubKey2].TargetToSource[0], "Attestations not merged correctly")
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
		proposalHistory, err := store.ProposalHistoryForEpoch(context.Background(), key[:], 0)
		require.NoError(t, err, "Retrieving proposal history failed for public key %v", key)
		expectedProposals := expectedHistory.Proposals[key]
		require.DeepEqual(t, expectedProposals, proposalHistory, "Proposals are incorrect")
	}

	attestationHistory, err := store.AttestationHistoryForPubKeys(context.Background(), pubKeys)
	require.NoError(t, err, "Retrieving attestation history failed")
	for _, key := range pubKeys {
		expectedAttestations := expectedHistory.Attestations[key]
		require.Equal(t, expectedAttestations[0], attestationHistory[key].TargetToSource[0], "Attestations are incorrect")
	}
}
