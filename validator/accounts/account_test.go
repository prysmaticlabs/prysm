package accounts

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/validator/db"

	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"gopkg.in/urfave/cli.v2"
)

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
	directory := "foobar"
	_, _, err := CreateValidatorAccount(directory, "foobar")
	wantErrString := fmt.Sprintf("path %q does not exist", directory)
	if err == nil || err.Error() != wantErrString {
		t.Errorf("expected error not thrown, want: %v, got: %v", wantErrString, err)
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
	path, passphrase, err := HandleEmptyFlags(ctx, false)
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

func TestMerge(t *testing.T) {
	firstSourcePubKeys := [][48]byte{{1}, {2}}
	firstStore := db.SetupDB(t, firstSourcePubKeys)
	secondSourcePubKeys := [][48]byte{{3}, {4}}
	secondStore := db.SetupDB(t, secondSourcePubKeys)

	proposalEpoch := uint64(0)
	proposalHistory1 := bitfield.Bitlist{0x01, 0x00, 0x00, 0x00, 0x01}
	if err := firstStore.SaveProposalHistoryForEpoch(context.Background(), firstSourcePubKeys[0][:], proposalEpoch, proposalHistory1); err != nil {
		t.Fatalf("Saving proposal history failed: %v", err)
	}
	proposalHistory2 := bitfield.Bitlist{0x02, 0x00, 0x00, 0x00, 0x01}
	if err := firstStore.SaveProposalHistoryForEpoch(context.Background(), firstSourcePubKeys[1][:], proposalEpoch, proposalHistory2); err != nil {
		t.Fatalf("Saving proposal history failed: %v", err)
	}
	proposalHistory3 := bitfield.Bitlist{0x03, 0x00, 0x00, 0x00, 0x01}
	if err := secondStore.SaveProposalHistoryForEpoch(context.Background(), secondSourcePubKeys[0][:], proposalEpoch, proposalHistory3); err != nil {
		t.Fatalf("Saving proposal history failed: %v", err)
	}
	proposalHistory4 := bitfield.Bitlist{0x04, 0x00, 0x00, 0x00, 0x01}
	if err := secondStore.SaveProposalHistoryForEpoch(context.Background(), secondSourcePubKeys[1][:], proposalEpoch, proposalHistory4); err != nil {
		t.Fatalf("Saving proposal history failed: %v", err)
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
	dbAttestationHistory1[firstSourcePubKeys[0]] = pubKeyAttestationHistory1
	dbAttestationHistory1[firstSourcePubKeys[1]] = pubKeyAttestationHistory2
	if err := firstStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory1); err != nil {
		t.Fatalf("Saving attestation history failed: %v", err)
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
	dbAttestationHistory2[secondSourcePubKeys[0]] = pubKeyAttestationHistory3
	dbAttestationHistory2[secondSourcePubKeys[1]] = pubKeyAttestationHistory4
	if err := secondStore.SaveAttestationHistoryForPubKeys(context.Background(), dbAttestationHistory2); err != nil {
		t.Fatalf("Saving attestation history failed: %v", err)
	}

	if err := firstStore.Close(); err != nil {
		t.Fatalf("Closing source store failed: %v", err)
	}
	if err := secondStore.Close(); err != nil {
		t.Fatalf("Closing source store failed: %v", err)
	}

	targetDirectory := testutil.TempDir() + "/target"
	err := Merge(context.Background(), []string{firstStore.DatabasePath(), secondStore.DatabasePath()}, targetDirectory)
	if err != nil {
		t.Fatalf("Merging failed: %v", err)
	}

	mergedStore, err := db.GetKVStore(targetDirectory)
	if err != nil {
		t.Fatalf("Retrieving the merged store failed: %v", err)
	}

	mergedProposalHistory1, err := mergedStore.ProposalHistoryForEpoch(context.Background(), firstSourcePubKeys[0][:], proposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", firstSourcePubKeys[0])
	} else {
		if !bytes.Equal(mergedProposalHistory1, proposalHistory1) {
			t.Errorf("Proposals not merged correctly: expected %v vs received %v", proposalHistory1, mergedProposalHistory1)
		}
	}
	mergedProposalHistory2, err := mergedStore.ProposalHistoryForEpoch(context.Background(), firstSourcePubKeys[1][:], proposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", firstSourcePubKeys[1])
	} else {
		if !bytes.Equal(mergedProposalHistory2, proposalHistory2) {
			t.Errorf("Proposals not merged correctly: expected %v vs received %v", proposalHistory2, mergedProposalHistory2)
		}
	}
	mergedProposalHistory3, err := mergedStore.ProposalHistoryForEpoch(context.Background(), secondSourcePubKeys[0][:], proposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", secondSourcePubKeys[0])
	} else {
		if !bytes.Equal(mergedProposalHistory3, proposalHistory3) {
			t.Errorf("Proposals not merged correctly: expected %v vs received %v", proposalHistory3, mergedProposalHistory3)
		}
	}
	mergedProposalHistory4, err := mergedStore.ProposalHistoryForEpoch(context.Background(), secondSourcePubKeys[1][:], proposalEpoch)
	if err != nil {
		t.Errorf("Retrieving merged proposal history failed for public key %v", secondSourcePubKeys[1])
	} else {
		if !bytes.Equal(mergedProposalHistory4, proposalHistory4) {
			t.Errorf("Proposals not merged correctly: expected %v vs received %v", proposalHistory4, mergedProposalHistory4)
		}
	}

	mergedAttestationHistory, err := mergedStore.AttestationHistoryForPubKeys(
		context.Background(),
		append(firstSourcePubKeys, secondSourcePubKeys[0], secondSourcePubKeys[1]))
	if err != nil {
		t.Error("Retrieving merged attestation history failed")
	} else {
		if mergedAttestationHistory[firstSourcePubKeys[0]].TargetToSource[0] != attestationHistoryMap1[0] {
			t.Errorf("Attestations not merged correctly: expected %v vs received %v", 0, mergedAttestationHistory[firstSourcePubKeys[0]].TargetToSource[0])
		}
		if mergedAttestationHistory[firstSourcePubKeys[1]].TargetToSource[0] != attestationHistoryMap2[0] {
			t.Errorf("Attestations not merged correctly: expected %v vs received %v", 0, mergedAttestationHistory[firstSourcePubKeys[1]].TargetToSource[0])
		}
		if mergedAttestationHistory[secondSourcePubKeys[0]].TargetToSource[0] != attestationHistoryMap3[0] {
			t.Errorf("Attestations not merged correctly: expected %v vs received %v", 0, mergedAttestationHistory[secondSourcePubKeys[0]].TargetToSource[0])
		}
		if mergedAttestationHistory[secondSourcePubKeys[1]].TargetToSource[0] != attestationHistoryMap4[0] {
			t.Errorf("Attestations not merged correctly: expected %v vs received %v", 0, mergedAttestationHistory[secondSourcePubKeys[1]].TargetToSource[0])
		}
	}
}

/*func TestMerge_SomeEmptyDirs(){

}

func TestMerge_AllEmptyDirs(){

}*/
