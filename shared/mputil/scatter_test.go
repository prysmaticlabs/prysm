package mputil_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/mputil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDouble(t *testing.T) {
	tests := []struct {
		name     string
		inValues int
		err      error
	}{
		{
			name:     "0",
			inValues: 0,
			err:      errors.New("input length must be greater than 0"),
		},
		{
			name:     "1",
			inValues: 1,
		},
		{
			name:     "1023",
			inValues: 1023,
		},
		{
			name:     "1024",
			inValues: 1024,
		},
		{
			name:     "1025",
			inValues: 1025,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inValues := make([]int, test.inValues)
			for i := 0; i < test.inValues; i++ {
				inValues[i] = i
			}
			outValues := make([]int, test.inValues)
			workerResults, err := mputil.Scatter(len(inValues), func(offset int, entries int, _ *sync.RWMutex) (interface{}, error) {
				extent := make([]int, entries)
				for i := 0; i < entries; i++ {
					extent[i] = inValues[offset+i] * 2
				}
				return extent, nil
			})
			if test.err != nil {
				if err == nil {
					t.Fatalf("Missing expected error %v", test.err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("Unexpected error value: expected \"%v\", found \"%v\"", test.err, err)
				}
			} else {
				require.NoError(t, err)
				for _, result := range workerResults {
					copy(outValues[result.Offset:], result.Extent.([]int))
				}

				for i := 0; i < test.inValues; i++ {
					if outValues[i] != inValues[i]*2 {
						t.Fatalf("Outvalue at %d mismatch: expected %d, found %d", i, inValues[i]*2, outValues[i])
					}
				}
			}
		})
	}
}

func TestMutex(t *testing.T) {
	totalRuns := 1048576
	val := 0
	_, err := mputil.Scatter(totalRuns, func(offset int, entries int, mu *sync.RWMutex) (interface{}, error) {
		for i := 0; i < entries; i++ {
			mu.Lock()
			val++
			mu.Unlock()
		}
		return nil, nil
	})
	require.NoError(t, err)

	if val != totalRuns {
		t.Fatalf("Unexpected value: expected \"%v\", found \"%v\"", totalRuns, val)
	}
}

func TestError(t *testing.T) {
	totalRuns := 1024
	val := 0
	_, err := mputil.Scatter(totalRuns, func(offset int, entries int, mu *sync.RWMutex) (interface{}, error) {
		for i := 0; i < entries; i++ {
			mu.Lock()
			val++
			if val == 1011 {
				mu.Unlock()
				return nil, errors.New("bad number")
			}
			mu.Unlock()
		}
		return nil, nil
	})
	if err == nil {
		t.Fatalf("Missing expected error")
	}
}
