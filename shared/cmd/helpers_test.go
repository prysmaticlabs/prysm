package cmd

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestEnterPassword(t *testing.T) {
	type rets struct {
		pw  string
		err error
	}
	var tt = []struct {
		name        string
		rets        []rets
		expectedErr error
		expectedPw  string
	}{
		{
			"first_match",
			[]rets{{"abcd", nil}, {"abcd", nil}},
			nil,
			"abcd",
		},
		{
			"first_match_with_newline",
			[]rets{{"abcd\n", nil}, {"abcd", nil}},
			nil,
			"abcd",
		},
		{
			"first_match_with_newline_confirm",
			[]rets{{"abcd", nil}, {"abcd\n", nil}},
			nil,
			"abcd",
		},
		{
			"first_match_both_newline",
			[]rets{{"abcd\n", nil}, {"abcd\n", nil}},
			nil,
			"abcd",
		},
		{
			"second_match",
			[]rets{{"abcd", nil}, {"aba", nil}, {"abcd", nil}, {"abcd", nil}},
			nil,
			"abcd",
		},
		{
			"cant_read",
			[]rets{{"pw", errors.New("i/o fail")}},
			errors.New("i/o fail"),
			"",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockPasswordReader(ctrl)
			for _, ret := range tc.rets {
				m.EXPECT().ReadPassword().Return(ret.pw, ret.err)
			}
			pw, err := EnterPassword(true, m)
			assert.Equal(t, tc.expectedPw, pw)
			if tc.expectedErr != nil {
				assert.ErrorContains(t, tc.expectedErr.Error(), err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
