package testdata

import (
	"math/rand"
	mathRand "math/rand"
	"time"
)

func UseRandNew() {
	randGenerator := mathRand.New(rand.NewSource(time.Now().UnixNano()))
	start := uint64(randGenerator.Intn(32))
	_ = start

	randGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func UseWithoutSeed() {
	assignedIndex := rand.Intn(int(128))
	_ = assignedIndex
}
