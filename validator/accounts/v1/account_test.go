package accounts

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
		assert.NoError(t, os.RemoveAll(directory))
	}()
	validatorKey, err := keystore.NewKey()
	require.NoError(t, err, "Cannot create new key")
	ks := keystore.NewKeystore(directory)
	err = ks.StoreKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, validatorKey, "")
	require.NoError(t, err, "Unable to store key")
	require.NoError(t, NewValidatorAccount(directory, "passsword123"), "Should support multiple keys")
	files, err := ioutil.ReadDir(directory)
	assert.NoError(t, err)
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
		assert.ErrorContains(t, wantErrString, err)
	})

	t.Run("empty existing dir", func(t *testing.T) {
		directory := testutil.TempDir() + "/testkeystore"
		defer func() {
			if err := os.RemoveAll(directory); err != nil {
				t.Logf("Could not remove directory: %v", err)
			}
		}()

		// Make sure that empty existing directory doesn't trigger any errors.
		require.NoError(t, os.Mkdir(directory, 0777))
		_, _, err := CreateValidatorAccount(directory, "foobar")
		assert.NoError(t, err)
	})

	t.Run("empty string as password", func(t *testing.T) {
		directory := testutil.TempDir() + "/testkeystore"
		defer func() {
			if err := os.RemoveAll(directory); err != nil {
				t.Logf("Could not remove directory: %v", err)
			}
		}()
		require.NoError(t, os.Mkdir(directory, 0777))
		_, _, err := CreateValidatorAccount(directory, "")
		wantErrString := "empty passphrase is not allowed"
		assert.ErrorContains(t, wantErrString, err)
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
	require.NoError(t, err)

	assert.Equal(t, passedPath, path, "Expected set path to be unchanged")
	assert.Equal(t, passedPassword, passphrase, "Expected set password to be unchanged")
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
	require.NoError(t, err, "Cannot create new key")
	ks := keystore.NewKeystore(directory)
	err = ks.StoreKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, validatorKey, oldPassword)
	require.NoError(t, err, "Unable to store key")

	require.NoError(t, ChangePassword(directory, oldPassword, newPassword))

	keys, err := DecryptKeysFromKeystore(directory, params.BeaconConfig().ValidatorPrivkeyFileName, newPassword)
	require.NoError(t, err)

	_, ok := keys[hex.EncodeToString(validatorKey.PublicKey.Marshal())]
	assert.Equal(t, true, ok, "Key not encrypted using the new password")
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
	require.NoError(t, err, "Cannot create new key")
	ks := keystore.NewKeystore(directory)
	err = ks.StoreKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, validatorKey, "notmatching")
	require.NoError(t, err, "Unable to store key")

	require.NoError(t, ChangePassword(directory, oldPassword, newPassword))

	keys, err := DecryptKeysFromKeystore(directory, params.BeaconConfig().ValidatorPrivkeyFileName, newPassword)
	require.NoError(t, err)
	_, ok := keys[hex.EncodeToString(validatorKey.PublicKey.Marshal())]
	assert.Equal(t, false, ok, "Key incorrectly encrypted using the new password")
}

func TestMerge_SucceedsWhenNoDatabaseExistsInSomeSourceDirectory(t *testing.T) {
	firstStorePubKey := [48]byte{1}
	firstStore := dbTest.SetupDB(t, [][48]byte{firstStorePubKey})
	secondStorePubKey := [48]byte{2}
	secondStore := dbTest.SetupDB(t, [][48]byte{secondStorePubKey})
	history, err := prepareSourcesForMerging(firstStorePubKey, firstStore.(*kv.Store), secondStorePubKey, secondStore.(*kv.Store))
	require.NoError(t, err)

	require.NoError(t, firstStore.Close(), "Closing source store failed")
	require.NoError(t, secondStore.Close(), "Closing source store failed")

	sourceDirectoryWithoutStore := testutil.TempDir() + "/nodb"
	require.NoError(t, os.MkdirAll(sourceDirectoryWithoutStore, 0700), "Could not create directory")
	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(targetDirectory), "Could not remove target directory")
	})

	err = Merge(
		context.Background(),
		[]string{firstStore.DatabasePath(), secondStore.DatabasePath(), sourceDirectoryWithoutStore}, targetDirectory)
	require.NoError(t, err, "Merging failed")
	mergedStore, err := kv.GetKVStore(targetDirectory)
	require.NoError(t, err, "Retrieving the merged store failed")

	assertMergedStore(t, mergedStore, firstStorePubKey, secondStorePubKey, history)
}

func TestMerge_FailsWhenNoDatabaseExistsInAllSourceDirectories(t *testing.T) {
	sourceDirectory1 := testutil.TempDir() + "/source1"
	sourceDirectory2 := testutil.TempDir() + "/source2"
	targetDirectory := testutil.TempDir() + "/target"
	require.NoError(t, os.MkdirAll(sourceDirectory1, 0700), "Could not create directory")
	require.NoError(t, os.MkdirAll(sourceDirectory2, 0700), "Could not create directory")
	require.NoError(t, os.MkdirAll(targetDirectory, 0700), "Could not create directory")
	t.Cleanup(func() {
		for _, dir := range []string{sourceDirectory1, sourceDirectory2, targetDirectory} {
			assert.NoError(t, os.RemoveAll(dir), "Could not remove directory")
		}
	})

	err := Merge(context.Background(), []string{sourceDirectory1, sourceDirectory2}, targetDirectory)
	expected := "no validator databases found in source directories"
	assert.ErrorContains(t, expected, err)
}

func TestSplit(t *testing.T) {
	pubKeys := [][48]byte{{1}, {2}}
	sourceStore := dbTest.SetupDB(t, pubKeys)

	proposalEpoch := uint64(0)
	proposalHistory1 := bitfield.Bitlist{0x01, 0x00, 0x00, 0x00, 0x01}
	err := sourceStore.SaveProposalHistoryForEpoch(context.Background(), pubKeys[0][:], proposalEpoch, proposalHistory1)
	require.NoError(t, err, "Saving proposal history failed")
	proposalHistory2 := bitfield.Bitlist{0x02, 0x00, 0x00, 0x00, 0x01}
	err = sourceStore.SaveProposalHistoryForEpoch(context.Background(), pubKeys[1][:], proposalEpoch, proposalHistory2)
	require.NoError(t, err, "Saving proposal history failed")

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
	err = sourceStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory)
	require.NoError(t, err, "Saving attestation history failed %v")

	require.NoError(t, sourceStore.Close(), "Closing source store failed")

	targetDirectory := testutil.TempDir() + "/target"
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(targetDirectory), "Could not remove target directory")
	})

	require.NoError(t, Split(context.Background(), sourceStore.DatabasePath(), targetDirectory), "Splitting failed")
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
	require.NoError(t, err, "Retrieving merged proposal history failed for public key %v", firstStorePubKey)
	require.DeepEqual(t, history.FirstStorePubKeyProposals, mergedProposalHistory1, "Proposals not merged correctly")
	mergedProposalHistory2, err := mergedStore.ProposalHistoryForEpoch(
		context.Background(), secondStorePubKey[:], history.ProposalEpoch)
	require.NoError(t, err, "Retrieving merged proposal history failed for public key %v", secondStorePubKey)
	require.DeepEqual(t, history.SecondStorePubKeyProposals, mergedProposalHistory2, "Proposals not merged correctly")

	mergedAttestationHistory, err := mergedStore.AttestationHistoryForPubKeys(
		context.Background(),
		[][48]byte{firstStorePubKey, secondStorePubKey})
	require.NoError(t, err, "Retrieving merged attestation history failed")
	assert.Equal(t, history.FirstStorePubKeyAttestations[0], mergedAttestationHistory[firstStorePubKey].TargetToSource[0])
	assert.Equal(t, history.SecondStorePubKeyAttestations[0], mergedAttestationHistory[secondStorePubKey].TargetToSource[0])
}
