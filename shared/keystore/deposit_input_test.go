package keystore_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
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

	sig, err := bls.SignatureFromBytes(result.Signature)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the proof of possession is a signed copy of the input data.
	proofOfPossessionInputPb := proto.Clone(result).(*pb.DepositData)
	proofOfPossessionInputPb.Signature = nil
	buf, err := ssz.Marshal(proofOfPossessionInputPb)
	if err != nil {
		t.Fatal(err)
	}
	dom := bytesutil.FromBytes8(params.BeaconConfig().DomainDeposit)
	if !sig.Verify(buf, k1.PublicKey, dom) {
		t.Error("Invalid proof of proofOfPossession signature")
	}
}
