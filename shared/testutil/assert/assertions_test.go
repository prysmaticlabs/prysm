package assert

import (
	"errors"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assertions"
)

func TestAssert_Equal(t *testing.T) {
	type args struct {
		tb       *assertions.TBMock
		expected interface{}
		actual   interface{}
		msgs     []interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedErr string
	}{
		{
			name: "equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   42,
			},
		},
		{
			name: "equal values different types",
			args: args{
				tb:       &assertions.TBMock{},
				expected: uint64(42),
				actual:   42,
			},
			expectedErr: "Values are not equal, want: 42 (uint64), got: 42 (int)",
		},
		{
			name: "non-equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   41,
			},
			expectedErr: "Values are not equal, want: 42 (int), got: 41 (int)",
		},
		{
			name: "custom error message",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   41,
				msgs:     []interface{}{"Custom values are not equal"},
			},
			expectedErr: "Custom values are not equal, want: 42 (int), got: 41 (int)",
		},
		{
			name: "custom error message with params",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   41,
				msgs:     []interface{}{"Custom values are not equal (for slot %d)", 12},
			},
			expectedErr: "Custom values are not equal (for slot 12), want: 42 (int), got: 41 (int)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Equal(tt.args.tb, tt.args.expected, tt.args.actual, tt.args.msgs...)
			if !strings.Contains(tt.args.tb.ErrorfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.ErrorfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_NotEqual(t *testing.T) {
	type args struct {
		tb       *assertions.TBMock
		expected interface{}
		actual   interface{}
		msgs     []interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedErr string
	}{
		{
			name: "equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   42,
			},
			expectedErr: "Values are equal, both values are equal: 42 (int)",
		},
		{
			name: "equal values different types",
			args: args{
				tb:       &assertions.TBMock{},
				expected: uint64(42),
				actual:   42,
			},
		},
		{
			name: "non-equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   41,
			},
		},
		{
			name: "custom error message",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   42,
				msgs:     []interface{}{"Custom values are equal"},
			},
			expectedErr: "Custom values are equal, both values are equal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NotEqual(tt.args.tb, tt.args.expected, tt.args.actual, tt.args.msgs...)
			if !strings.Contains(tt.args.tb.ErrorfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.ErrorfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_DeepEqual(t *testing.T) {
	type args struct {
		tb       *assertions.TBMock
		expected interface{}
		actual   interface{}
		msgs     []interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedErr string
	}{
		{
			name: "equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ i int }{42},
				actual:   struct{ i int }{42},
			},
		},
		{
			name: "non-equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ i int }{42},
				actual:   struct{ i int }{41},
			},
			expectedErr: "Values are not equal, want: {42}, got: {41}",
		},
		{
			name: "custom error message",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ i int }{42},
				actual:   struct{ i int }{41},
				msgs:     []interface{}{"Custom values are not equal"},
			},
			expectedErr: "Custom values are not equal, want: {42}, got: {41}",
		},
		{
			name: "custom error message with params",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ i int }{42},
				actual:   struct{ i int }{41},
				msgs:     []interface{}{"Custom values are not equal (for slot %d)", 12},
			},
			expectedErr: "Custom values are not equal (for slot 12), want: {42}, got: {41}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeepEqual(tt.args.tb, tt.args.expected, tt.args.actual, tt.args.msgs...)
			if !strings.Contains(tt.args.tb.ErrorfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.ErrorfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_NoError(t *testing.T) {
	type args struct {
		tb   *assertions.TBMock
		err  error
		msgs []interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedErr string
	}{
		{
			name: "nil error",
			args: args{
				tb: &assertions.TBMock{},
			},
		},
		{
			name: "non-nil error",
			args: args{
				tb:  &assertions.TBMock{},
				err: errors.New("failed"),
			},
			expectedErr: "Unexpected error: failed",
		},
		{
			name: "custom non-nil error",
			args: args{
				tb:   &assertions.TBMock{},
				err:  errors.New("failed"),
				msgs: []interface{}{"Custom error message"},
			},
			expectedErr: "Custom error message: failed",
		},
		{
			name: "custom non-nil error with params",
			args: args{
				tb:   &assertions.TBMock{},
				err:  errors.New("failed"),
				msgs: []interface{}{"Custom error message (for slot %d)", 12},
			},
			expectedErr: "Custom error message (for slot 12): failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NoError(tt.args.tb, tt.args.err, tt.args.msgs...)
			if !strings.Contains(tt.args.tb.ErrorfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.ErrorfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_ErrorContains(t *testing.T) {
	type args struct {
		tb   *assertions.TBMock
		want string
		err  error
		msgs []interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedErr string
	}{
		{
			name: "nil error",
			args: args{
				tb:   &assertions.TBMock{},
				want: "some error",
			},
			expectedErr: "Expected error not returned, got: <nil>, want: some error",
		},
		{
			name: "unexpected error",
			args: args{
				tb:   &assertions.TBMock{},
				want: "another error",
				err:  errors.New("failed"),
			},
			expectedErr: "Expected error not returned, got: failed, want: another error",
		},
		{
			name: "expected error",
			args: args{
				tb:   &assertions.TBMock{},
				want: "failed",
				err:  errors.New("failed"),
			},
			expectedErr: "",
		},
		{
			name: "custom unexpected error",
			args: args{
				tb:   &assertions.TBMock{},
				want: "another error",
				err:  errors.New("failed"),
				msgs: []interface{}{"Something wrong"},
			},
			expectedErr: "Something wrong, got: failed, want: another error",
		},
		{
			name: "expected error",
			args: args{
				tb:   &assertions.TBMock{},
				want: "failed",
				err:  errors.New("failed"),
				msgs: []interface{}{"Something wrong"},
			},
			expectedErr: "",
		},
		{
			name: "custom unexpected error with params",
			args: args{
				tb:   &assertions.TBMock{},
				want: "another error",
				err:  errors.New("failed"),
				msgs: []interface{}{"Something wrong (for slot %d)", 12},
			},
			expectedErr: "Something wrong (for slot 12), got: failed, want: another error",
		},
		{
			name: "expected error with params",
			args: args{
				tb:   &assertions.TBMock{},
				want: "failed",
				err:  errors.New("failed"),
				msgs: []interface{}{"Something wrong (for slot %d)", 12},
			},
			expectedErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ErrorContains(tt.args.tb, tt.args.want, tt.args.err, tt.args.msgs...)
			if !strings.Contains(tt.args.tb.ErrorfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.ErrorfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_NotNil(t *testing.T) {
	type args struct {
		tb   *assertions.TBMock
		obj  interface{}
		msgs []interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedErr string
	}{
		{
			name: "nil",
			args: args{
				tb: &assertions.TBMock{},
			},
			expectedErr: "Unexpected nil value",
		},
		{
			name: "nil custom message",
			args: args{
				tb:   &assertions.TBMock{},
				msgs: []interface{}{"This should not be nil"},
			},
			expectedErr: "This should not be nil",
		},
		{
			name: "nil custom message with params",
			args: args{
				tb:   &assertions.TBMock{},
				msgs: []interface{}{"This should not be nil (for slot %d)", 12},
			},
			expectedErr: "This should not be nil (for slot 12)",
		},
		{
			name: "not nil",
			args: args{
				tb:  &assertions.TBMock{},
				obj: "some value",
			},
			expectedErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NotNil(tt.args.tb, tt.args.obj, tt.args.msgs...)
			if !strings.Contains(tt.args.tb.ErrorfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.ErrorfMsg, tt.expectedErr)
			}
		})
	}
}
