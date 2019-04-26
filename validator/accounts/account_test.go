package accounts

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

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
	if err := NewValidatorAccount(directory, ""); err != nil {
		t.Errorf("Should support multiple keys: %v", err)
	}
	files, _ := ioutil.ReadDir(directory)
	if len(files) != 3 {
		t.Errorf("multiple validators were not created only %v files in directory", len(files))
		for _, f := range files {
			t.Errorf("%v\n", f.Name())
		}
	}
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("Could not remove directory: %v", err)
	}
}
