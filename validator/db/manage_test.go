package db

import (
	"bytes"
	"context"
	"encoding/hex"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

type sourceStoresHistory struct {
	ProposalEpoch                       uint64
	FirstStoreFirstPubKeyProposals      bitfield.Bitlist
	FirstStoreSecondPubKeyProposals     bitfield.Bitlist
	SecondStoreFirstPubKeyProposals     bitfield.Bitlist
	SecondStoreSecondPubKeyProposals    bitfield.Bitlist
	FirstStoreFirstPubKeyAttestations   map[uint64]uint64
	FirstStoreSecondPubKeyAttestations  map[uint64]uint64
	SecondStoreFirstPubKeyAttestations  map[uint64]uint64
	SecondStoreSecondPubKeyAttestations map[uint64]uint64
}

func TestMerge(t *testing.T) {
	firstStorePubKeys := [][48]byte{{1}, {2}}
	firstStore := SetupDB(t, firstStorePubKeys)
	secondStorePubKeys := [][48]byte{{3}, {4}}
	secondStore := SetupDB(t, secondStorePubKeys)

	history, err := prepareSourcesForMerging(firstStorePubKeys, firstStore, secondStorePubKeys, secondStore)
	if err != nil {
		t.Fatal(err)
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

	assertMergedStore(t, mergedStore, firstStorePubKeys, secondStorePubKeys, history)
}

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
	dbAttestationHistory1 := make(map[[48]byte]*slashpb.AttestationHistory)
	dbAttestationHistory1[firstStorePubKeys[0]] = pubKeyAttestationHistory1
	dbAttestationHistory1[firstStorePubKeys[1]] = pubKeyAttestationHistory2
	if err := firstStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory1); err != nil {
		return nil, errors.Wrapf(err, "Saving attestation history failed")
	}
	attestationHistoryMap3 := make(map[uint64]uint64)
	attestationHistoryMap3[0] = 2
	pubKeyAttestationHistory3 := &slashpb.AttestationHistory{
		TargetToSource:     attestationHistoryMap3,
		LatestEpochWritten: 0,
	}
	attestationHistoryMap4 := make(map[uint64]uint64)
	attestationHistoryMap4[0] = 3
	pubKeyAttestationHistory4 := &slashpb.AttestationHistory{
		TargetToSource:     attestationHistoryMap4,
		LatestEpochWritten: 0,
	}
	dbAttestationHistory2 := make(map[[48]byte]*slashpb.AttestationHistory)
	dbAttestationHistory2[secondStorePubKeys[0]] = pubKeyAttestationHistory3
	dbAttestationHistory2[secondStorePubKeys[1]] = pubKeyAttestationHistory4
	if err := secondStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory2); err != nil {
		return nil, errors.Wrapf(err, "Saving attestation history failed")
	}

	mergeHistory := &sourceStoresHistory{
		ProposalEpoch:                       proposalEpoch,
		FirstStoreFirstPubKeyProposals:      proposalHistory1,
		FirstStoreSecondPubKeyProposals:     proposalHistory2,
		SecondStoreFirstPubKeyProposals:     proposalHistory3,
		SecondStoreSecondPubKeyProposals:    proposalHistory4,
		FirstStoreFirstPubKeyAttestations:   attestationHistoryMap1,
		FirstStoreSecondPubKeyAttestations:  attestationHistoryMap2,
		SecondStoreFirstPubKeyAttestations:  attestationHistoryMap3,
		SecondStoreSecondPubKeyAttestations: attestationHistoryMap4,
	}

	return mergeHistory, nil
}

	func prepareSourcesForMerging(
		firstStorePubKeys [][48]byte,
		firstStore *Store,
		secondStorePubKeys [][48]byte,
		secondStore *Store) (*sourceStoresHistory, error) {

		proposalEpoch := uint64(0)
		proposalHistory1 := bitfield.Bitlist{0x01, 0x00, 0x00, 0x00, 0x01}
		if err := firstStore.SaveProposalHistoryForEpoch(context.Background(), firstStorePubKeys[0][:], proposalEpoch, proposalHistory1); err != nil {
			return nil, errors.Wrapf(err, "Saving proposal history failed")
		}
		proposalHistory2 := bitfield.Bitlist{0x02, 0x00, 0x00, 0x00, 0x01}
		if err := firstStore.SaveProposalHistoryForEpoch(context.Background(), firstStorePubKeys[1][:], proposalEpoch, proposalHistory2); err != nil {
			return nil, errors.Wrapf(err, "Saving proposal history failed")
		}
		proposalHistory3 := bitfield.Bitlist{0x03, 0x00, 0x00, 0x00, 0x01}
		if err := secondStore.SaveProposalHistoryForEpoch(context.Background(), secondStorePubKeys[0][:], proposalEpoch, proposalHistory3); err != nil {
			return nil, errors.Wrapf(err, "Saving proposal history failed")
		}
		proposalHistory4 := bitfield.Bitlist{0x04, 0x00, 0x00, 0x00, 0x01}
		if err := secondStore.SaveProposalHistoryForEpoch(context.Background(), secondStorePubKeys[1][:], proposalEpoch, proposalHistory4); err != nil {
			return nil, errors.Wrapf(err, "Saving proposal history failed")
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
		dbAttestationHistory1 := make(map[[48]byte]*slashpb.AttestationHistory)
		dbAttestationHistory1[firstStorePubKeys[0]] = pubKeyAttestationHistory1
		dbAttestationHistory1[firstStorePubKeys[1]] = pubKeyAttestationHistory2
		if err := firstStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory1); err != nil {
			return nil, errors.Wrapf(err, "Saving attestation history failed")
		}
		attestationHistoryMap3 := make(map[uint64]uint64)
		attestationHistoryMap3[0] = 2
		pubKeyAttestationHistory3 := &slashpb.AttestationHistory{
			TargetToSource:     attestationHistoryMap3,
			LatestEpochWritten: 0,
		}
		attestationHistoryMap4 := make(map[uint64]uint64)
		attestationHistoryMap4[0] = 3
		pubKeyAttestationHistory4 := &slashpb.AttestationHistory{
			TargetToSource:     attestationHistoryMap4,
			LatestEpochWritten: 0,
		}
		dbAttestationHistory2 := make(map[[48]byte]*slashpb.AttestationHistory)
		dbAttestationHistory2[secondStorePubKeys[0]] = pubKeyAttestationHistory3
		dbAttestationHistory2[secondStorePubKeys[1]] = pubKeyAttestationHistory4
		if err := secondStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory2); err != nil {
			return nil, errors.Wrapf(err, "Saving attestation history failed")
		}

		mergeHistory := &sourceStoresHistory{
			ProposalEpoch:                       proposalEpoch,
			FirstStoreFirstPubKeyProposals:      proposalHistory1,
			FirstStoreSecondPubKeyProposals:     proposalHistory2,
			SecondStoreFirstPubKeyProposals:     proposalHistory3,
			SecondStoreSecondPubKeyProposals:    proposalHistory4,
			FirstStoreFirstPubKeyAttestations:   attestationHistoryMap1,
			FirstStoreSecondPubKeyAttestations:  attestationHistoryMap2,
			SecondStoreFirstPubKeyAttestations:  attestationHistoryMap3,
			SecondStoreSecondPubKeyAttestations: attestationHistoryMap4,
		}

		return mergeHistory, nil
	}

func assertMergedStore(
	t *testing.T,
	mergedStore *Store,
	firstStorePubKeys [][48]byte,
	secondStorePubKeys [][48]byte,
	history *sourceStoresHistory) {

	mergedProposalHistory1, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), firstStorePubKeys[0][:], history.ProposalEpoch)
	if err != nil {
		t.Fatalf("Retrieving merged proposal history failed for public key %v", firstStorePubKeys[0])
	}
	if !bytes.Equal(mergedProposalHistory1, history.FirstStoreFirstPubKeyProposals) {
		t.Fatalf(
			"Proposals not merged correctly: expected %v vs received %v",
			history.FirstStoreFirstPubKeyProposals,
			mergedProposalHistory1)
	}
	mergedProposalHistory2, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), firstStorePubKeys[1][:], history.ProposalEpoch)
	if err != nil {
		t.Fatalf("Retrieving merged proposal history failed for public key %v", firstStorePubKeys[1])
	}
	if !bytes.Equal(mergedProposalHistory2, history.FirstStoreSecondPubKeyProposals) {
		t.Fatalf(
			"Proposals not merged correctly: expected %v vs received %v",
			history.FirstStoreSecondPubKeyProposals,
			mergedProposalHistory2)
	}
	mergedProposalHistory3, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), secondStorePubKeys[0][:], history.ProposalEpoch)
	if err != nil {
		t.Fatalf("Retrieving merged proposal history failed for public key %v", secondStorePubKeys[0])
	}
	if !bytes.Equal(mergedProposalHistory3, history.SecondStoreFirstPubKeyProposals) {
		t.Fatalf(
			"Proposals not merged correctly: expected %v vs received %v",
			history.SecondStoreFirstPubKeyProposals,
			mergedProposalHistory3)
	}
	mergedProposalHistory4, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), secondStorePubKeys[1][:], history.ProposalEpoch)
	if err != nil {
		t.Fatalf("Retrieving merged proposal history failed for public key %v", secondStorePubKeys[1])
	}
	if !bytes.Equal(mergedProposalHistory4, history.SecondStoreSecondPubKeyProposals) {
		t.Fatalf("Proposals not merged correctly: expected %v vs received %v",
			history.SecondStoreSecondPubKeyProposals,
			mergedProposalHistory4)
	}

	mergedAttestationHistory, err := mergedStore.AttestationHistoryForPubKeys(
		context.Background(),
		append(firstStorePubKeys, secondStorePubKeys[0], secondStorePubKeys[1]))
	if err != nil {
		t.Fatalf("Retrieving merged attestation history failed")
	}
	if mergedAttestationHistory[firstStorePubKeys[0]].TargetToSource[0] != history.FirstStoreFirstPubKeyAttestations[0] {
		t.Fatalf(
			"Attestations not merged correctly: expected %v vs received %v",
			history.FirstStoreFirstPubKeyAttestations[0],
			mergedAttestationHistory[firstStorePubKeys[0]].TargetToSource[0])
	}
	if mergedAttestationHistory[firstStorePubKeys[1]].TargetToSource[0] != history.FirstStoreSecondPubKeyAttestations[0] {
		t.Fatalf(
			"Attestations not merged correctly: expected %v vs received %v",
			history.FirstStoreSecondPubKeyAttestations,
			mergedAttestationHistory[firstStorePubKeys[1]].TargetToSource[0])
	}
	if mergedAttestationHistory[secondStorePubKeys[0]].TargetToSource[0] != history.SecondStoreFirstPubKeyAttestations[0] {
		t.Fatalf(
			"Attestations not merged correctly: expected %v vs received %v",
			history.SecondStoreFirstPubKeyAttestations,
			mergedAttestationHistory[secondStorePubKeys[0]].TargetToSource[0])
	}
	if mergedAttestationHistory[secondStorePubKeys[1]].TargetToSource[0] != history.SecondStoreSecondPubKeyAttestations[0] {
		t.Fatalf(
			"Attestations not merged correctly: expected %v vs received %v",
			history.SecondStoreSecondPubKeyAttestations,
			mergedAttestationHistory[secondStorePubKeys[1]].TargetToSource[0])
	}
}
