package accounts

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
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

func TestNewValidatorAccount_PrintsDepositData(t *testing.T) {
	hook := logTest.NewGlobal()
	directory := testutil.TempDir() + "/testkeystore"
	defer os.RemoveAll(directory)
	if err := NewValidatorAccount(directory, "1234"); err != nil {
		t.Errorf("Expected new account to be created: %v", err)
	}
	ks := keystore.NewKeystore(directory)
	valKey, err := ks.GetKey(directory+params.BeaconConfig().ValidatorPrivkeyFileName, "1234")
	if err != nil {
		t.Fatalf("Could not retrieve key: %v", err)
	}
	data := &pb.DepositInput{
		Pubkey:                      valKey.SecretKey.GetPublicKey().Serialize(),
		ProofOfPossession:           []byte("pop"),
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize deposit data: %v", err)
	}
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("%#x", serializedData))
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("Could not remove directory: %v", err)
	}
}
