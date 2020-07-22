package v1

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestPathsToVerificationRegexes(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		regexes []string
		err     string
	}{
		{
			name:    "Empty",
			regexes: []string{},
		},
		{
			name:    "IgnoreBadPaths",
			paths:   []string{"", "/", "/Account"},
			regexes: []string{},
		},
		{
			name:    "Simple",
			paths:   []string{"Wallet/Account"},
			regexes: []string{"^Wallet/Account$"},
		},
		{
			name:    "Multiple",
			paths:   []string{"Wallet/Account1", "Wallet/Account2"},
			regexes: []string{"^Wallet/Account1$", "^Wallet/Account2$"},
		},
		{
			name:    "IgnoreInvalidRegex",
			paths:   []string{"Wallet/Account1", "Bad/***", "Wallet/Account2"},
			regexes: []string{"^Wallet/Account1$", "^Wallet/Account2$"},
		},
		{
			name:    "TidyExistingAnchors",
			paths:   []string{"Wallet/^.*$", "Wallet/Foo.*Bar$", "Wallet/^Account"},
			regexes: []string{"^Wallet/.*$", "^Wallet/Foo.*Bar$", "^Wallet/Account$"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			regexes := pathsToVerificationRegexes(test.paths)
			require.Equal(t, len(test.regexes), len(regexes), "Unexpected number of regexes")
			for i := range regexes {
				require.Equal(t, test.regexes[i], regexes[i].String(), "Unexpected regex %d", i)
			}
		})
	}
}
