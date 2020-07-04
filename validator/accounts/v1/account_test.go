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
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

type sourceStoresHistory struct {
	ProposalEpoch                 uint64
	FirstStorePubKeyProposals     bitfield.Bitlist
	SecondStorePubKeyProposals    bitfield.Bitlist
	FirstStorePubKeyAttestations  map[uint64]uint64
	SecondStorePubKeyAttestations map[uint64]uint64
}

func TestNewValidatorAccount_AccountExists(t *testing.T) {
	directory := testutil.TempDir() + "/testkeystore"
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("Could not remove directory: %v", err)
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
	if err := NewValidatorAccount(directory, "passsword123"); err != nil {
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
	t.Run("custom non-existent path", func(t *testing.T) {
		_, _, err := CreateValidatorAccount("foobar", "foobar")
		wantErrString := fmt.Sprintf("path %q does not exist", "foobar")
		if err == nil || err.Error() != wantErrString {
			t.Errorf("expected error not thrown, want: %v, got: %v", wantErrString, err)
		}
	})

	t.Run("empty existing dir", func(t *testing.T) {
		directory := testutil.TempDir() + "/testkeystore"
		defer func() {
			if err := os.RemoveAll(directory); err != nil {
				t.Logf("Could not remove directory: %v", err)
			}
		}()

		// Make sure that empty existing directory doesn't trigger any errors.
		if err := os.Mkdir(directory, 0777); err != nil {
			t.Fatal(err)
		}
		_, _, err := CreateValidatorAccount(directory, "foobar")
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("empty string as password", func(t *testing.T) {
		directory := testutil.TempDir() + "/testkeystore"
		defer func() {
			if err := os.RemoveAll(directory); err != nil {
				t.Logf("Could not remove directory: %v", err)
			}
		}()
		if err := os.Mkdir(directory, 0777); err != nil {
			t.Fatal(err)
		}
		_, _, err := CreateValidatorAccount(directory, "")
		wantErrString := "empty passphrase is not allowed"
		if err == nil || !strings.Contains(err.Error(), wantErrString) {
			t.Errorf("expected error not thrown, want: %v, got: %v", wantErrString, err)
		}
	})
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

func TestMerge_SucceedsWhenNoDatabaseExistsInSomeSourceDirectory(t *testing.T) {
	firstStorePubKey := [48]byte{1}
	firstStore := dbTest.SetupDB(t, [][48]byte{firstStorePubKey})
	secondStorePubKey := [48]byte{2}
	secondStore := dbTest.SetupDB(t, [][48]byte{secondStorePubKey})
	history, err := prepareSourcesForMerging(firstStorePubKey, firstStore.(*kv.Store), secondStorePubKey, secondStore.(*kv.Store))
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
	t.Cleanup(func() {
		if err := os.RemoveAll(targetDirectory); err != nil {
			t.Errorf("Could not remove target directory : %v", err)
		}
	})

	err = Merge(
		context.Background(),
		[]string{firstStore.DatabasePath(), secondStore.DatabasePath(), sourceDirectoryWithoutStore}, targetDirectory)
	if err != nil {
		t.Fatalf("Merging failed: %v", err)
	}
	mergedStore, err := kv.GetKVStore(targetDirectory)
	if err != nil {
		t.Fatalf("Retrieving the merged store failed: %v", err)
	}

	assertMergedStore(t, mergedStore, firstStorePubKey, secondStorePubKey, history)
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
	t.Cleanup(func() {
		for _, dir := range []string{sourceDirectory1, sourceDirectory2, targetDirectory} {
			if err := os.RemoveAll(dir); err != nil {
				t.Errorf("Could not remove directory : %v", err)
			}
		}
	})

	err := Merge(context.Background(), []string{sourceDirectory1, sourceDirectory2}, targetDirectory)
	expected := "no validator databases found in source directories"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected: %s vs received %v", expected, err)
	}
}

func TestSplit(t *testing.T) {
	pubKeys := [][48]byte{{1}, {2}}
	sourceStore := dbTest.SetupDB(t, pubKeys)

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

	if err := sourceStore.Close(); err != nil {
		t.Fatalf("Closing source store failed: %v", err)
	}

	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		if err := os.RemoveAll(targetDirectory); err != nil {
			t.Errorf("Could not remove target directory : %v", err)
		}
	})

	if err := Split(context.Background(), sourceStore.DatabasePath(), targetDirectory); err != nil {
		t.Fatalf("Splitting failed: %v", err)
	}
}

func prepareSourcesForMerging(firstStorePubKey [48]byte, firstStore *kv.Store, secondStorePubKey [48]byte, secondStore *kv.Store) (*sourceStoresHistory, error) {
	proposalEpoch := uint64(0)
	proposalHistory1 := bitfield.Bitlist{0x01, 0x00, 0x00, 0x00, 0x01}
	if err := firstStore.SaveProposalHistoryForEpoch(context.Background(), firstStorePubKey[:], proposalEpoch, proposalHistory1); err != nil {
		return nil, errors.Wrapf(err, "Saving proposal history failed")
	}
	proposalHistory2 := bitfield.Bitlist{0x02, 0x00, 0x00, 0x00, 0x01}
	if err := secondStore.SaveProposalHistoryForEpoch(context.Background(), secondStorePubKey[:], proposalEpoch, proposalHistory2); err != nil {
		return nil, errors.Wrapf(err, "Saving proposal history failed")
	}

	attestationHistoryMap1 := make(map[uint64]uint64)
	attestationHistoryMap1[0] = 0
	pubKeyAttestationHistory1 := &slashpb.AttestationHistory{
		TargetToSource:     attestationHistoryMap1,
		LatestEpochWritten: 0,
	}
	dbAttestationHistory1 := make(map[[48]byte]*slashpb.AttestationHistory)
	dbAttestationHistory1[firstStorePubKey] = pubKeyAttestationHistory1
	if err := firstStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory1); err != nil {
		return nil, errors.Wrapf(err, "Saving attestation history failed")
	}
	attestationHistoryMap2 := make(map[uint64]uint64)
	attestationHistoryMap2[0] = 1
	pubKeyAttestationHistory2 := &slashpb.AttestationHistory{
		TargetToSource:     attestationHistoryMap2,
		LatestEpochWritten: 0,
	}
	dbAttestationHistory2 := make(map[[48]byte]*slashpb.AttestationHistory)
	dbAttestationHistory2[secondStorePubKey] = pubKeyAttestationHistory2
	if err := secondStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory2); err != nil {
		return nil, errors.Wrapf(err, "Saving attestation history failed")
	}

	mergeHistory := &sourceStoresHistory{
		ProposalEpoch:                 proposalEpoch,
		FirstStorePubKeyProposals:     proposalHistory1,
		SecondStorePubKeyProposals:    proposalHistory2,
		FirstStorePubKeyAttestations:  attestationHistoryMap1,
		SecondStorePubKeyAttestations: attestationHistoryMap2,
	}

	return mergeHistory, nil
}

func assertMergedStore(
	t *testing.T,
	mergedStore *kv.Store,
	firstStorePubKey [48]byte,
	secondStorePubKey [48]byte,
	history *sourceStoresHistory) {

	mergedProposalHistory1, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), firstStorePubKey[:], history.ProposalEpoch)
	if err != nil {
		t.Fatalf("Retrieving merged proposal history failed for public key %v", firstStorePubKey)
	}
	if !bytes.Equal(mergedProposalHistory1, history.FirstStorePubKeyProposals) {
		t.Fatalf(
			"Proposals not merged correctly: expected %v vs received %v",
			history.FirstStorePubKeyProposals,
			mergedProposalHistory1)
	}
	mergedProposalHistory2, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), secondStorePubKey[:], history.ProposalEpoch)
	if err != nil {
		t.Fatalf("Retrieving merged proposal history failed for public key %v", secondStorePubKey)
	}
	if !bytes.Equal(mergedProposalHistory2, history.SecondStorePubKeyProposals) {
		t.Fatalf(
			"Proposals not merged correctly: expected %v vs received %v",
			history.SecondStorePubKeyProposals,
			mergedProposalHistory2)
	}

	mergedAttestationHistory, err := mergedStore.AttestationHistoryForPubKeys(
		context.Background(),
		[][48]byte{firstStorePubKey, secondStorePubKey})
	if err != nil {
		t.Fatalf("Retrieving merged attestation history failed")
	}
	if mergedAttestationHistory[firstStorePubKey].TargetToSource[0] != history.FirstStorePubKeyAttestations[0] {
		t.Fatalf(
			"Attestations not merged correctly: expected %v vs received %v",
			history.FirstStorePubKeyAttestations[0],
			mergedAttestationHistory[firstStorePubKey].TargetToSource[0])
	}
	if mergedAttestationHistory[secondStorePubKey].TargetToSource[0] != history.SecondStorePubKeyAttestations[0] {
		t.Fatalf(
			"Attestations not merged correctly: expected %v vs received %v",
			history.SecondStorePubKeyAttestations,
			mergedAttestationHistory[secondStorePubKey].TargetToSource[0])
	}
}
