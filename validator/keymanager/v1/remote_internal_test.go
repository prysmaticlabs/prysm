package v1

import (
	"testing"
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
			if len(regexes) != len(test.regexes) {
				t.Fatalf("Unexpected number of regexes: expected %v, received %v", len(test.regexes), len(regexes))
			}
			for i := range regexes {
				if regexes[i].String() != test.regexes[i] {
					t.Fatalf("Unexpected regex %d: expected %v, received %v", i, test.regexes[i], regexes[i].String())
				}
			}
		})
	}
}
