package require

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
		msg      []string
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
			name: "non-equal values",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   41,
			},
			expectedErr: "Values are not equal, got: 41, want: 42",
		},
		{
			name: "custom error message",
			args: args{
				tb:       &assertions.TBMock{},
				expected: 42,
				actual:   41,
				msg:      []string{"Custom values are not equal"},
			},
			expectedErr: "Custom values are not equal, got: 41, want: 42",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Equal(tt.args.tb, tt.args.expected, tt.args.actual, tt.args.msg...)
			if !strings.Contains(tt.args.tb.FatalfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.FatalfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_DeepEqual(t *testing.T) {
	type args struct {
		tb       *assertions.TBMock
		expected interface{}
		actual   interface{}
		msg      []string
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
			expectedErr: "Values are not equal, got: {41}, want: {42}",
		},
		{
			name: "custom error message",
			args: args{
				tb:       &assertions.TBMock{},
				expected: struct{ i int }{42},
				actual:   struct{ i int }{41},
				msg:      []string{"Custom values are not equal"},
			},
			expectedErr: "Custom values are not equal, got: {41}, want: {42}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeepEqual(tt.args.tb, tt.args.expected, tt.args.actual, tt.args.msg...)
			if !strings.Contains(tt.args.tb.FatalfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.FatalfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_NoError(t *testing.T) {
	type args struct {
		tb  *assertions.TBMock
		err error
		msg []string
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
			name: "non-nil error",
			args: args{
				tb:  &assertions.TBMock{},
				err: errors.New("failed"),
				msg: []string{"Custom error message"},
			},
			expectedErr: "Custom error message: failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NoError(tt.args.tb, tt.args.err, tt.args.msg...)
			if !strings.Contains(tt.args.tb.FatalfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.FatalfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_ErrorContains(t *testing.T) {
	type args struct {
		tb   *assertions.TBMock
		want string
		err  error
		msg  []string
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
				msg:  []string{"Something wrong"},
			},
			expectedErr: "Something wrong, got: failed, want: another error",
		},
		{
			name: "expected error",
			args: args{
				tb:   &assertions.TBMock{},
				want: "failed",
				err:  errors.New("failed"),
				msg:  []string{"Something wrong"},
			},
			expectedErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ErrorContains(tt.args.tb, tt.args.want, tt.args.err, tt.args.msg...)
			if !strings.Contains(tt.args.tb.FatalfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.FatalfMsg, tt.expectedErr)
			}
		})
	}
}

func TestAssert_NotNil(t *testing.T) {
	type args struct {
		tb  *assertions.TBMock
		obj interface{}
		msg []string
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
				tb:  &assertions.TBMock{},
				msg: []string{"This should not be nil"},
			},
			expectedErr: "This should not be nil",
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
			NotNil(tt.args.tb, tt.args.obj, tt.args.msg...)
			if !strings.Contains(tt.args.tb.FatalfMsg, tt.expectedErr) {
				t.Errorf("got: %q, want: %q", tt.args.tb.FatalfMsg, tt.expectedErr)
			}
		})
	}
}
