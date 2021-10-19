package testdata

import (
	"math/rand"
	mathRand "math/rand"
	"time"
)

// UseRandNew --
func UseRandNew() {
	source := rand.NewSource(time.Now().UnixNano()) // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	randGenerator := mathRand.New(source)           // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	start := uint64(randGenerator.Intn(32))
	_ = start

	source = rand.NewSource(time.Now().UnixNano()) // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	randGenerator = rand.New(source)               // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
}

// UseWithoutSeed --
func UseWithoutSeed() {
	assignedIndex := rand.Intn(128) // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	_ = assignedIndex
}
