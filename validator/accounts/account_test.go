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
