package attaggregation

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
		AttestationAggregationStrategy: "naive",
	})
	defer resetCfg()
	os.Exit(m.Run())
}

func bitlistWithAllBitsSet(length uint64) bitfield.Bitlist {
	b := bitfield.NewBitlist(length)
	for i := uint64(0); i < length; i++ {
		b.SetBitAt(i, true)
	}
	return b
}

func bitlistsWithSingleBitSet(length uint64) []bitfield.Bitlist {
	lists := make([]bitfield.Bitlist, length)
	for i := uint64(0); i < length; i++ {
		b := bitfield.NewBitlist(length)
		b.SetBitAt(i, true)
		lists[i] = b
	}
	return lists
}

func bitlistsWithMultipleBitSet(length uint64, count uint64) []bitfield.Bitlist {
	rand.Seed(time.Now().UnixNano())
	lists := make([]bitfield.Bitlist, length)
	for i := uint64(0); i < length; i++ {
		b := bitfield.NewBitlist(length)
		keys := rand.Perm(int(length))
		for _, key := range keys[:count] {
			b.SetBitAt(uint64(key), true)
		}
		lists[i] = b
	}
	return lists
}

func makeAttestationsFromBitlists(bl []bitfield.Bitlist) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, len(bl))
	for i, b := range bl {
		atts[i] = &ethpb.Attestation{
			AggregationBits: b,
			Data:            nil,
			Signature:       bls.NewAggregateSignature().Marshal(),
		}
	}
	return atts
}
