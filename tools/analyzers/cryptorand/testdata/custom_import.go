package testdata

import (
	mathRand "math/rand"
	xxx "math/rand"
	"time"
)

func UseRandNewCustomImport() {
	randGenerator := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	start := uint64(randGenerator.Intn(32))
	_ = start

	randGenerator = mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
}

func UseWithoutSeeCustomImportd() {
	assignedIndex := mathRand.Intn(128)
	_ = assignedIndex
	xxx.Shuffle(10, func(i, j int) {

	})
}
