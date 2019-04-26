package keystore_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

func TestDepositInput_GeneratesPb(t *testing.T) {
	k1, err := keystore.NewKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := keystore.NewKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	result, err := keystore.DepositInput(k1, k2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Pubkey, k1.PublicKey.Marshal()) {
		t.Errorf("Mismatched pubkeys in deposit input. Want = %x, got = %x", result.Pubkey, k1.PublicKey.Marshal())
	}

	sig, err := bls.SignatureFromBytes(result.ProofOfPossession)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the proof of possession is a signed copy of the input data.
	proofOfPossessionInputPb := proto.Clone(result).(*pb.DepositInput)
	proofOfPossessionInputPb.ProofOfPossession = nil
	buf := new(bytes.Buffer)
	if err := ssz.Encode(buf, proofOfPossessionInputPb); err != nil {
		t.Fatal(err)
	}

	if !sig.Verify(buf.Bytes(), k1.PublicKey, params.BeaconConfig().DomainDeposit) {
		t.Error("Invalid proof of proofOfPossession signature")
	}
}
