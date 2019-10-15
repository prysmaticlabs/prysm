package mputil_test

import (
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/mputil"
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
			workers, outValuesCh, _, err := mputil.Scatter(len(inValues), func(offset int, entries int) (*mputil.ScatterResults, error) {
				extent := make([]int, entries)
				result := mputil.NewScatterResults(offset, extent)
				for i := 0; i < entries; i++ {
					extent[i] = inValues[offset+i] * 2
				}
				return result, nil
			})
			if test.err != nil {
				if err == nil {
					t.Fatalf("Missing expected error %v", test.err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("Unexpected error value: expected \"%v\", found \"%v\"", test.err, err)
				}
			}

			for i := workers; i > 0; i-- {
				result := <-outValuesCh
				copy(outValues[result.Offset:], result.Extent.([]int))
			}

			for i := 0; i < test.inValues; i++ {
				if outValues[i] != inValues[i]*2 {
					t.Fatalf("Outvalue at %d mismatch: expected %d, found %d", i, inValues[i]*2, outValues[i])
				}
			}
		})
	}
}
