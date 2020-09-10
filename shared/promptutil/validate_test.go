package promptutil

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestValidatePasswordInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantedErr string
	}{
		{
			name:      "no numbers nor special characters",
			input:     "abcdefghijklmnopqrs",
			wantedErr: "password must have more than 8 characters, at least 1 special character, and 1 number",
		},
		{
			name:      "number and letters but no special characters",
			input:     "abcdefghijklmnopqrs2020",
			wantedErr: "password must have more than 8 characters, at least 1 special character, and 1 number",
		},
		{
			name:      "numbers, letters, special characters, but too short",
			input:     "abc2$",
			wantedErr: "password must have more than 8 characters, at least 1 special character, and 1 number",
		},
		{
			name:  "proper length and strong password",
			input: "%Str0ngpassword32kjAjsd22020$%",
		},
		{
			name:      "password format correct but weak entropy score",
			input:     "aaaaaaa1$",
			wantedErr: "password is too easy to guess, try a stronger password",
		},
		{
			name:  "allow spaces",
			input: "x*329293@aAJSD i22903saj",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordInput(tt.input)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidUnicode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Regular alphanumeric",
			input: "Someone23xx",
			want:  true,
		},
		{
			name:  "Unicode strings separated by a space character",
			input: "x*329293@aAJSD i22903saj",
			want:  true,
		},
		{
			name:  "Japanese",
			input: "僕は絵お見るのが好きです",
			want:  true,
		},
		{
			name:  "Other foreign",
			input: "Etérium",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUnicode(tt.input); got != tt.want {
				t.Errorf("isValidUnicode() = %v, want %v", got, tt.want)
			}
		})
	}
}
