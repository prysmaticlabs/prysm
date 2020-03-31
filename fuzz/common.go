package fuzz

import (
	"strings"

	"github.com/prysmaticlabs/go-ssz"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

const PanicOnError = "false"
const ReturnSSZPostState = false

func init() {
	featureconfig.Init(&featureconfig.Flags{
		SkipBLSVerify: true,
	})
}

func fail(err error) ([]byte, bool) {
	if strings.ToLower(PanicOnError) == "true" {
		panic(err)
	}
	return nil, false
}

func success(post *stateTrie.BeaconState) ([]byte, bool) {
	if !ReturnSSZPostState {
		return nil, true
	}

	result, err := ssz.Marshal(post.InnerStateUnsafe())
	if err != nil {
		panic(err)
	}
	return result, true
}