package shared

import (
	"testing"
)

type numbertTest struct {
	number uint64
	root   uint64
}

func TestIntegerSquareRoot(t *testing.T) {
	tt := []numbertTest{
		numbertTest{
			number: 20,
			root:   4,
		},
		numbertTest{
			number: 200,
			root:   14,
		},
		numbertTest{
			number: 1987,
			root:   44,
		},
		numbertTest{
			number: 34989843,
			root:   5915,
		},
		numbertTest{
			number: 97282,
			root:   311,
		},
	}

	for _, testVals := range tt {
		root := IntegerSquareRoot(testVals.number)
		if testVals.root != root {
			t.Fatalf("expected root and computed root are not equal %d, %d", testVals.root, root)
		}
	}
}
