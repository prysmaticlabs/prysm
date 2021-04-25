package testdata

import (
	foobar "math/rand"
	mathRand "math/rand"
	"time"
)

// UseRandNewCustomImport --
func UseRandNewCustomImport() {
	source := mathRand.NewSource(time.Now().UnixNano()) // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	randGenerator := mathRand.New(source)               // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	start := uint64(randGenerator.Intn(32))
	_ = start

	source = mathRand.NewSource(time.Now().UnixNano()) // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	randGenerator = mathRand.New(source)               // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
}

// UseWithoutSeeCustomImport --
func UseWithoutSeeCustomImport() {
	assignedIndex := mathRand.Intn(128) // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"
	_ = assignedIndex
	foobar.Shuffle(10, func(i, j int) { // want "crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand"

	})
}
