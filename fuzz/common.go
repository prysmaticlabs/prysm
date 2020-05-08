package fuzz

import (
	"os"
	"strings"

	"github.com/prysmaticlabs/go-ssz"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func init() {
	featureconfig.Init(&featureconfig.Flags{
		SkipBLSVerify: true,
	})
}

func fail(err error) ([]byte, bool) {
	shouldPanic := false
	if val, ok := os.LookupEnv("PANIC_ON_ERROR"); ok {
		shouldPanic = strings.ToLower(val) == "true"
	}
	if shouldPanic {
		panic(err)
	}
	return nil, false
}

func success(post *stateTrie.BeaconState) ([]byte, bool) {
	if val, ok := os.LookupEnv("RETURN_SSZ_POST_STATE"); ok {
		if strings.ToLower(val) != "true" {
			return nil, true
		}
	}

	result, err := ssz.Marshal(post.InnerStateUnsafe())
	if err != nil {
		panic(err)
	}
	return result, true
}
