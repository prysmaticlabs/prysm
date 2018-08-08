// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"golang.org/x/crypto/blake2b"
)

// ShuffleIndices returns a list of pseudorandomly sampled
// indices. This is used to use to select attesters and proposers.
func ShuffleIndices(seed common.Hash, validatorCount int) ([]int, error) {
	if validatorCount > params.MaxValidators {
		return nil, errors.New("Validator count has exceeded MaxValidator Count")
	}

	// construct a list of indices up to MaxValidators.
	validatorList := make([]int, validatorCount)
	for i := range validatorList {
		validatorList[i] = i
	}

	hashSeed := blake2b.Sum256(seed[:])

	// shuffle stops at the second to last index.
	for i := 0; i < validatorCount-1; i++ {
		// convert every 3 bytes to random number, replace validator index with that number.
		for j := 0; j+3 < len(hashSeed); j += 3 {
			swapNum := int(hashSeed[j] + hashSeed[j+1] + hashSeed[j+2])
			remaining := validatorCount - i
			swapPos := swapNum%remaining + i
			validatorList[i], validatorList[swapPos] = validatorList[swapPos], validatorList[i]
		}
	}
	return validatorList, nil
}

// GetCutoffs is used to split up validators into groups at the start
// of every epoch. It determines at what height validators can make
// attestations and crosslinks. It returns lists of cutoff indices.
func GetCutoffs(validatorCount int) []int {
	var heightCutoff = []int{0}
	var heights []int
	var heightCount float64

	// Skip heights if there's not enough validators to fill in a min sized committee.
	if validatorCount < params.EpochLength*params.MinCommiteeSize {
		heightCount = math.Floor(float64(validatorCount) / params.MinCommiteeSize)
		for i := 0; i < int(heightCount); i++ {
			heights = append(heights, (i*params.Cofactor)%params.EpochLength)
		}
		// Enough validators, fill in all the heights.
	} else {
		heightCount = params.EpochLength
		for i := 0; i < int(heightCount); i++ {
			heights = append(heights, i)
		}
	}

	filled := 0
	appendHeight := false
	for i := 0; i < params.EpochLength-1; i++ {
		appendHeight = false
		for _, height := range heights {
			if i == height {
				appendHeight = true
			}
		}
		if appendHeight {
			filled++
			heightCutoff = append(heightCutoff, filled*validatorCount/int(heightCount))
		} else {
			heightCutoff = append(heightCutoff, heightCutoff[len(heightCutoff)-1])
		}
	}
	heightCutoff = append(heightCutoff, validatorCount)

	// TODO: For the validators assigned to each height, split them up into
	// committees for different shards. Do not assign the last END_EPOCH_GRACE_PERIOD
	// heights in a epoch to any shards.
	return heightCutoff
}
