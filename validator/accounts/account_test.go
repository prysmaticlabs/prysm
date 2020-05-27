package accounts

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"gopkg.in/urfave/cli.v2"
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

func TestNewValidatorAccount_AccountExists(t *testing.T) {
	directory := testutil.TempDir() + "/testkeystore"
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Logf("Could not remove directory: %v", err)
		}
	}()
	validatorKey, err := keystore.NewKey()
	if err != nil {
		t.Fatalf("Cannot create new key: %v", err)
	}
	ks := keystore.NewKeystore(directory)
	if err := ks.StoreKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, validatorKey, ""); err != nil {
		t.Fatalf("Unable to store key %v", err)
	}
	if err := NewValidatorAccount(directory, ""); err != nil {
		t.Errorf("Should support multiple keys: %v", err)
	}
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		t.Error(err)
	}
	if len(files) != 3 {
		t.Errorf("multiple validators were not created only %v files in directory", len(files))
		for _, f := range files {
			t.Errorf("%v\n", f.Name())
		}
	}
}

func TestNewValidatorAccount_CreateValidatorAccount(t *testing.T) {
	directory := testutil.TempDir() + "/testkeystore"
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Logf("Could not remove directory: %v", err)
		}
	}()
	_, _, err := CreateValidatorAccount("foobar", "foobar")
	wantErrString := fmt.Sprintf("path %q does not exist", "foobar")
	if err == nil || err.Error() != wantErrString {
		t.Errorf("expected error not thrown, want: %v, got: %v", wantErrString, err)
	}

	// Make sure that empty existing directory doesn't trigger any errors.
	if err := os.Mkdir(directory, 0777); err != nil {
		t.Fatal(err)
	}
	_, _, err = CreateValidatorAccount(directory, "foobar")
	if err != nil {
		t.Error(err)
	}
}

func TestHandleEmptyFlags_FlagsSet(t *testing.T) {
	passedPath := "~/path/given"
	passedPassword := "password"

	app := &cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.KeystorePathFlag.Name, passedPath, "set keystore path")
	set.String(flags.PasswordFlag.Name, passedPassword, "set keystore password")
	ctx := cli.NewContext(app, set, nil)
	path, passphrase, err := HandleEmptyKeystoreFlags(ctx, false)
	if err != nil {
		t.Fatal(err)
	}

	if passedPath != path {
		t.Fatalf("Expected set path to be unchanged, expected %s, received %s", passedPath, path)
	}
	if passedPassword != passphrase {
		t.Fatalf("Expected set password to be unchanged, expected %s, received %s", passedPassword, passphrase)
	}
}

func TestChangePassword_KeyEncryptedWithNewPassword(t *testing.T) {
	directory := testutil.TempDir() + "/testkeystore"
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Logf("Could not remove directory: %v", err)
		}
	}()

	oldPassword := "old"
	newPassword := "new"

	validatorKey, err := keystore.NewKey()
	if err != nil {
		t.Fatalf("Cannot create new key: %v", err)
	}
	ks := keystore.NewKeystore(directory)
	if err := ks.StoreKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, validatorKey, oldPassword); err != nil {
		t.Fatalf("Unable to store key %v", err)
	}

	if err := ChangePassword(directory, oldPassword, newPassword); err != nil {
		t.Fatal(err)
	}

	keys, err := DecryptKeysFromKeystore(directory, params.BeaconConfig().ValidatorPrivkeyFileName, newPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := keys[hex.EncodeToString(validatorKey.PublicKey.Marshal())]; !ok {
		t.Error("Key not encrypted using the new password")
	}
}

func TestChangePassword_KeyNotMatchingOldPasswordNotEncryptedWithNewPassword(t *testing.T) {
	directory := testutil.TempDir() + "/testkeystore"
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Logf("Could not remove directory: %v", err)
		}
	}()

	oldPassword := "old"
	newPassword := "new"

	validatorKey, err := keystore.NewKey()
	if err != nil {
		t.Fatalf("Cannot create new key: %v", err)
	}
	ks := keystore.NewKeystore(directory)
	if err := ks.StoreKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, validatorKey, "notmatching"); err != nil {
		t.Fatalf("Unable to store key %v", err)
	}

	if err := ChangePassword(directory, oldPassword, newPassword); err != nil {
		t.Fatal(err)
	}

	keys, err := DecryptKeysFromKeystore(directory, params.BeaconConfig().ValidatorPrivkeyFileName, newPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := keys[hex.EncodeToString(validatorKey.PublicKey.Marshal())]; ok {
		t.Error("Key incorrectly encrypted using the new password")
	}
}

func TestMerge(t *testing.T) {
	firstStorePubKeys := [][48]byte{{1}, {2}}
	firstStore := db.SetupDB(t, firstStorePubKeys)
	secondStorePubKeys := [][48]byte{{3}, {4}}
	secondStore := db.SetupDB(t, secondStorePubKeys)

	history, err := prepareSourcesForMerging(firstStorePubKeys, firstStore, secondStorePubKeys, secondStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if err := firstStore.Close(); err != nil {
		t.Fatalf("Closing source store failed: %v", err)
	}
	if err := secondStore.Close(); err != nil {
		t.Fatalf("Closing source store failed: %v", err)
	}

	targetDirectory := testutil.TempDir() + "/target"
	err = Merge(context.Background(), []string{firstStore.DatabasePath(), secondStore.DatabasePath()}, targetDirectory)
	if err != nil {
		t.Fatalf("Merging failed: %v", err)
	}
	mergedStore, err := db.GetKVStore(targetDirectory)
	if err != nil {
		t.Fatalf("Retrieving the merged store failed: %v", err)
	}

	assertMergedStore(t, mergedStore, firstStorePubKeys, secondStorePubKeys, history)

	cleanupAfterMerge(t, []string{firstStore.DatabasePath(), secondStore.DatabasePath(), targetDirectory})
}

func TestMerge_SucceedsWhenNoDatabaseExistsInSomeSourceDirectory(t *testing.T) {
	firstStorePubKeys := [][48]byte{{1}, {2}}
	firstStore := db.SetupDB(t, firstStorePubKeys)
	secondStorePubKeys := [][48]byte{{3}, {4}}
	secondStore := db.SetupDB(t, secondStorePubKeys)

	history, err := prepareSourcesForMerging(firstStorePubKeys, firstStore, secondStorePubKeys, secondStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if err := firstStore.Close(); err != nil {
		t.Fatalf("Closing source store failed: %v", err)
	}
	if err := secondStore.Close(); err != nil {
		t.Fatalf("Closing source store failed: %v", err)
	}

	sourceDirectoryWithoutStore := testutil.TempDir() + "/nodb"
	if err := os.MkdirAll(sourceDirectoryWithoutStore, 0700); err != nil {
		t.Fatalf("Could not create directory %s", sourceDirectoryWithoutStore)
	}
	targetDirectory := testutil.TempDir() + "/target"
	err = Merge(
		context.Background(),
		[]string{firstStore.DatabasePath(), secondStore.DatabasePath(), sourceDirectoryWithoutStore}, targetDirectory)
	if err != nil {
		t.Fatalf("Merging failed: %v", err)
	}
	mergedStore, err := db.GetKVStore(targetDirectory)
	if err != nil {
		t.Fatalf("Retrieving the merged store failed: %v", err)
	}

	assertMergedStore(t, mergedStore, firstStorePubKeys, secondStorePubKeys, history)

	cleanupAfterMerge(
		t,
		[]string{firstStore.DatabasePath(), secondStore.DatabasePath(), sourceDirectoryWithoutStore, targetDirectory})
}

func TestMerge_FailsWhenNoDatabaseExistsInAllSourceDirectories(t *testing.T) {
	sourceDirectory1 := testutil.TempDir() + "/source1"
	sourceDirectory2 := testutil.TempDir() + "/source2"
	targetDirectory := testutil.TempDir() + "/target"
	if err := os.MkdirAll(sourceDirectory1, 0700); err != nil {
		t.Fatalf("Could not create directory %s", sourceDirectory1)
	}
	if err := os.MkdirAll(sourceDirectory2, 0700); err != nil {
		t.Fatalf("Could not create directory %s", sourceDirectory2)
	}
	if err := os.MkdirAll(targetDirectory, 0700); err != nil {
		t.Fatalf("Could not create directory %s", targetDirectory)
	}

	err := Merge(context.Background(), []string{sourceDirectory1, sourceDirectory2}, targetDirectory)
	expected := "no validator databases found in source directories"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected: %s vs received %v", expected, err)
	}

	cleanupAfterMerge(t, []string{sourceDirectory1, sourceDirectory2, targetDirectory})
}

func prepareSourcesForMerging(firstStorePubKeys [][48]byte, firstStore *db.Store, secondStorePubKeys [][48]byte, secondStore *db.Store) (*sourceStoresHistory, error) {
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
	mergedStore *db.Store,
	firstStorePubKeys [][48]byte,
	secondStorePubKeys [][48]byte,
	history *sourceStoresHistory) {

	mergedProposalHistory1, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), firstStorePubKeys[0][:], history.ProposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", firstStorePubKeys[0])
	} else {
		if !bytes.Equal(mergedProposalHistory1, history.FirstStoreFirstPubKeyProposals) {
			t.Errorf(
				"Proposals not merged correctly: expected %v vs received %v",
				history.FirstStoreFirstPubKeyProposals,
				mergedProposalHistory1)
		}
	}
	mergedProposalHistory2, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), firstStorePubKeys[1][:], history.ProposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", firstStorePubKeys[1])
	} else {
		if !bytes.Equal(mergedProposalHistory2, history.FirstStoreSecondPubKeyProposals) {
			t.Errorf(
				"Proposals not merged correctly: expected %v vs received %v",
				history.FirstStoreSecondPubKeyProposals,
				mergedProposalHistory2)
		}
	}
	mergedProposalHistory3, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), secondStorePubKeys[0][:], history.ProposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", secondStorePubKeys[0])
	} else {
		if !bytes.Equal(mergedProposalHistory3, history.SecondStoreFirstPubKeyProposals) {
			t.Errorf(
				"Proposals not merged correctly: expected %v vs received %v",
				history.SecondStoreFirstPubKeyProposals,
				mergedProposalHistory3)
		}
	}
	mergedProposalHistory4, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), secondStorePubKeys[1][:], history.ProposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", secondStorePubKeys[1])
	} else {
		if !bytes.Equal(mergedProposalHistory4, history.SecondStoreSecondPubKeyProposals) {
			t.Errorf("Proposals not merged correctly: expected %v vs received %v",
				history.SecondStoreSecondPubKeyProposals,
				mergedProposalHistory4)
		}
	}

	mergedAttestationHistory, err := mergedStore.AttestationHistoryForPubKeys(
		context.Background(),
		append(firstStorePubKeys, secondStorePubKeys[0], secondStorePubKeys[1]))
	if err != nil {
		t.Error("Retrieving merged attestation history failed")
	} else {
		if mergedAttestationHistory[firstStorePubKeys[0]].TargetToSource[0] != history.FirstStoreFirstPubKeyAttestations[0] {
			t.Errorf(
				"Attestations not merged correctly: expected %v vs received %v",
				history.FirstStoreFirstPubKeyAttestations[0],
				mergedAttestationHistory[firstStorePubKeys[0]].TargetToSource[0])
		}
		if mergedAttestationHistory[firstStorePubKeys[1]].TargetToSource[0] != history.FirstStoreSecondPubKeyAttestations[0] {
			t.Errorf(
				"Attestations not merged correctly: expected %v vs received %v",
				history.FirstStoreSecondPubKeyAttestations,
				mergedAttestationHistory[firstStorePubKeys[1]].TargetToSource[0])
		}
		if mergedAttestationHistory[secondStorePubKeys[0]].TargetToSource[0] != history.SecondStoreFirstPubKeyAttestations[0] {
			t.Errorf(
				"Attestations not merged correctly: expected %v vs received %v",
				history.SecondStoreFirstPubKeyAttestations,
				mergedAttestationHistory[secondStorePubKeys[0]].TargetToSource[0])
		}
		if mergedAttestationHistory[secondStorePubKeys[1]].TargetToSource[0] != history.SecondStoreSecondPubKeyAttestations[0] {
			t.Errorf(
				"Attestations not merged correctly: expected %v vs received %v",
				history.SecondStoreSecondPubKeyAttestations,
				mergedAttestationHistory[secondStorePubKeys[1]].TargetToSource[0])
		}
	}
}

func cleanupAfterMerge(t *testing.T, directories []string) {
	for _, dir := range directories {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Could not remove directory %s: %v", dir, err)
		}
	}
}
