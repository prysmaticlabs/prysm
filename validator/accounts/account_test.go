package accounts

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

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

func TestMerge_KeysCopiedToNewDirectory(t *testing.T) {
	firstSourceDirectory := testutil.TempDir() + "/firstsource"
	secondSourceDirectory := testutil.TempDir() + "/secondsource"
	targetDirectory := testutil.TempDir() + "/target"
	firstSourcePassword := "firstsource"
	secondSourcePassword := "secondsource"
	targetPassword := "target"

	defer func() {
		if err := os.RemoveAll(firstSourceDirectory); err != nil {
			t.Logf("Could not remove directory %s: %v", firstSourceDirectory, err)
		}
		if err := os.RemoveAll(secondSourceDirectory); err != nil {
			t.Logf("Could not remove directory %s: %v", secondSourceDirectory, err)
		}
		if err := os.RemoveAll(targetDirectory); err != nil {
			t.Logf("Could not remove directory %s: %v", targetDirectory, err)
		}
	}()

	if err := NewValidatorAccount(firstSourceDirectory, firstSourcePassword); err != nil {
		t.Fatal(err)
	}
	if err := NewValidatorAccount(secondSourceDirectory, secondSourcePassword); err != nil {
		t.Fatal(err)
	}

	sourcePasswords := map[string]string{firstSourceDirectory: firstSourcePassword, secondSourceDirectory: secondSourcePassword}

	err := Merge(sourcePasswords, targetDirectory, targetPassword)
	if err != nil {
		t.Fatal(err)
	}

	firstSourceKeys, err := ioutil.ReadDir(firstSourceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	secondSourceKeys, err := ioutil.ReadDir(secondSourceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	targetKeys, err := ioutil.ReadDir(targetDirectory)
	if err != nil {
		t.Fatal(err)
	}

	for _, sk := range append(firstSourceKeys, secondSourceKeys...) {
		found := false
		for _, tk := range targetKeys {
			if sk.Name() == tk.Name() {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Key file %s not found", sk.Name())
		}
	}
}
