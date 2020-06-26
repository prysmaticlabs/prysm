package testdata

import (
	foobar "math/rand"
	mathRand "math/rand"
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
	foobar.Shuffle(10, func(i, j int) {

	})
}
