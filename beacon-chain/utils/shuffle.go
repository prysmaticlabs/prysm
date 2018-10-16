// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// ShuffleIndices returns a list of pseudorandomly sampled
// indices. This is used to use to select attesters and proposers.
func ShuffleIndices(seed common.Hash, validatorList []uint32) ([]uint32, error) {
	// Since we are consuming 3 bytes of entropy at a time in the loop,
	// we have a bias at 2**24, this check defines our max list size and is used to remove the bias.
	// more info on modulo bias: https://stackoverflow.com/questions/10984974/why-do-people-say-there-is-modulo-bias-when-using-a-random-number-generator.
	if len(validatorList) > params.GetConfig().ModuloBias {
		return nil, errors.New("exceeded upper bound for validator shuffle")
	}

	hashSeed := hashutil.Hash(seed[:])
	validatorCount := len(validatorList)

	// Shuffle stops at the second to last index.
	for i := 0; i < validatorCount-1; i++ {
		// Convert every 3 bytes to random number, replace validator index with that number.
		for j := 0; j+3 < len(hashSeed); j += 3 {
			swapNum := int(hashSeed[j] + hashSeed[j+1] + hashSeed[j+2])
			remaining := validatorCount - i
			swapPos := swapNum%remaining + i
			validatorList[i], validatorList[swapPos] = validatorList[swapPos], validatorList[i]
		}
		hashSeed = hashutil.Hash(seed[:])
	}
	return validatorList, nil
}

// SplitIndices splits a list into n pieces.
func SplitIndices(l []uint32, n int) [][]uint32 {
	var divided [][]uint32
	for i := 0; i < n; i++ {
		start := len(l) * i / n
		end := len(l) * (i + 1) / n
		divided = append(divided, l[start:end])
	}
	return divided
}
