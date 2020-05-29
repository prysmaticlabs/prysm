package db

import (
	"bytes"
	"context"
	"os"
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
