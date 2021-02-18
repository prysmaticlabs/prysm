package sszutil_test

import (
	"fmt"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sszutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assertions"
)

func TestAssert_DeepSSZEqual(t *testing.T) {
	type args struct {
		tb       *assertions.TBMock
		expected interface{}
		actual   interface{}
	}
	tests := []struct {
		name           string
		args           args
		expectedResult bool
	}{
		{
			name: "equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ I uint64 }{42},
				actual:   struct{ I uint64 }{42},
			},
			expectedResult: true,
		},
		{
			name: "equal structs",
			args: args{
				tb: &assertions.TBMock{},
				expected: &eth.Checkpoint{
					Epoch: 5,
					Root:  []byte("hi there"),
				},
				actual: &eth.Checkpoint{
					Epoch: 5,
					Root:  []byte("hi there"),
				},
			},
			expectedResult: true,
		},
		{
			name: "non-equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ I uint64 }{42},
				actual:   struct{ I uint64 }{41},
			},
			expectedResult: false,
		},
	}
	for _, tt := range tests {
		verify := func() {
			if tt.expectedResult && tt.args.tb.ErrorfMsg != "" {
				t.Errorf("Unexpected error: %s %v", tt.name, tt.args.tb.ErrorfMsg)
			}
		}
		t.Run(fmt.Sprintf("Assert/%s", tt.name), func(t *testing.T) {
			sszutil.AssertDeepEqual(tt.args.tb, tt.args.expected, tt.args.actual)
			verify()
		})
		t.Run(fmt.Sprintf("Require/%s", tt.name), func(t *testing.T) {
			sszutil.RequireDeepEqual(tt.args.tb, tt.args.expected, tt.args.actual)
			verify()
		})
	}
}

func TestAssert_DeepNotSSZEqual(t *testing.T) {
	type args struct {
		tb       *assertions.TBMock
		expected interface{}
		actual   interface{}
	}
	tests := []struct {
		name           string
		args           args
		expectedResult bool
	}{
		{
			name: "equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ I uint64 }{42},
				actual:   struct{ I uint64 }{42},
			},
			expectedResult: true,
		},
		{
			name: "non-equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ I uint64 }{42},
				actual:   struct{ I uint64 }{41},
			},
			expectedResult: false,
		},
		{
			name: "not equal structs",
			args: args{
				tb: &assertions.TBMock{},
				expected: &eth.Checkpoint{
					Epoch: 5,
					Root:  []byte("hello there"),
				},
				actual: &eth.Checkpoint{
					Epoch: 3,
					Root:  []byte("hi there"),
				},
			},
			expectedResult: true,
		},
	}
	for _, tt := range tests {
		verify := func() {
			if !tt.expectedResult && tt.args.tb.ErrorfMsg != "" {
				t.Errorf("Unexpected error: %s %v", tt.name, tt.args.tb.ErrorfMsg)
			}
		}
		t.Run(fmt.Sprintf("Assert/%s", tt.name), func(t *testing.T) {
			sszutil.AssertDeepNotEqual(tt.args.tb, tt.args.expected, tt.args.actual)
			verify()
		})
		t.Run(fmt.Sprintf("Require/%s", tt.name), func(t *testing.T) {
			sszutil.RequireDeepNotEqual(tt.args.tb, tt.args.expected, tt.args.actual)
			verify()
		})
	}
}
