//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"regexp"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

func validRoot(root string) bool {
	matchesRegex, err := regexp.MatchString("^0x[a-fA-F0-9]{64}$", root)
	if err != nil {
		return false
	}
	return matchesRegex
}

func getForkVersion(epoch types.Epoch) ([4]byte, error) {
	var forkVersionSlice []byte

	switch {
	case epoch < params.BeaconConfig().AltairForkEpoch:
		forkVersionSlice = params.BeaconConfig().GenesisForkVersion
	case epoch < params.BeaconConfig().BellatrixForkEpoch:
		forkVersionSlice = params.BeaconConfig().AltairForkVersion
	case epoch < params.BeaconConfig().CapellaForkEpoch:
		forkVersionSlice = params.BeaconConfig().BellatrixForkVersion
	case epoch < params.BeaconConfig().ShardingForkEpoch:
		forkVersionSlice = params.BeaconConfig().CapellaForkVersion
	default:
		forkVersionSlice = params.BeaconConfig().ShardingForkVersion
	}

	var forkVersion [4]byte
	if len(forkVersionSlice) != 4 {
		return forkVersion, errors.Errorf("invalid fork version: %s", hexutil.Encode(forkVersionSlice))
	}

	copy(forkVersion[:], forkVersionSlice)
	return forkVersion, nil
}
