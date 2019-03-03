package accounts

import (
	"crypto/rand"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestNewValidatorAccount_AccountExists(t *testing.T) {
	directory := testutil.TempDir() + "/testkeystore"
	defer os.RemoveAll(directory)
	validatorKey, err := keystore.NewKey(rand.Reader)
	if err != nil {
		t.Fatalf("Cannot create new key: %v", err)
	}
	ks := keystore.NewKeystore(directory)
	if err := ks.StoreKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, validatorKey, ""); err != nil {
		t.Fatalf("Unable to store key %v", err)
	}
	if err := NewValidatorAccount(directory, ""); err == nil {
		t.Error("Expected new validator account to throw error, received nil")
	}
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("Could not remove directory: %v", err)
	}
}
