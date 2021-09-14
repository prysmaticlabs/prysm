package promptutil

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
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
			wantedErr: errPasswordWeak.Error(),
		},
		{
			name:      "number and letters but no special characters",
			input:     "abcdefghijklmnopqrs2020",
			wantedErr: errPasswordWeak.Error(),
		},
		{
			name:      "numbers, letters, special characters, but too short",
			input:     "abc2$",
			wantedErr: errPasswordWeak.Error(),
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
		{
			name:  "strong password from LastPass",
			input: "jXl!q5pkQnXsyT6dbJ3X5plQ!9%iqJCTr&*UIoaDu#b6GYJD##^GI3qniKdr240f",
		},
		{
			name:  "allow underscores",
			input: "jXl!q5pkQn_syT6dbJ3X5plQ_9_iqJCTr_*UIoaDu#b6GYJD##^GI3qniKdr240f",
		},
		{
			name:      "only numbers and symbols should fail",
			input:     "123493489223423_23923929",
			wantedErr: errPasswordWeak.Error(),
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

func TestDefaultAndValidatePrompt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		def       string
		want      string
		wantError bool
	}{
		{
			name:  "number",
			input: "3",
			def:   "0",
			want:  "3",
		},
		{
			name:  "empty return default",
			input: "",
			def:   "0",
			want:  "0",
		},
		{
			name:  "empty return default no zero",
			input: "",
			def:   "3",
			want:  "3",
		},
		{
			name:      "empty return default, no zero",
			input:     "a",
			def:       "0",
			want:      "",
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte(tt.input + "\n")
			tmpfile, err := ioutil.TempFile("", "content")
			require.NoError(t, err)
			defer func() {
				err := os.Remove(tmpfile.Name())
				require.NoError(t, err)
			}()

			_, err = tmpfile.Write(content)
			require.NoError(t, err)

			_, err = tmpfile.Seek(0, 0)
			require.NoError(t, err)
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }() // Restore original Stdin
			os.Stdin = tmpfile
			got, err := DefaultAndValidatePrompt(tt.name, tt.def, ValidateNumber)
			if !tt.wantError {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
			err = tmpfile.Close()
			require.NoError(t, err)
		})
	}
}

func TestValidatePhrase(t *testing.T) {
	wantedPhrase := "wanted phrase"

	t.Run("correct input", func(t *testing.T) {
		assert.NoError(t, ValidatePhrase(wantedPhrase, wantedPhrase))
	})
	t.Run("correct input with whitespace", func(t *testing.T) {
		assert.NoError(t, ValidatePhrase("  wanted phrase  ", wantedPhrase))
	})
	t.Run("incorrect input", func(t *testing.T) {
		err := ValidatePhrase("foo", wantedPhrase)
		assert.NotNil(t, err)
		assert.ErrorContains(t, errIncorrectPhrase.Error(), err)
	})
	t.Run("wrong letter case", func(t *testing.T) {
		err := ValidatePhrase("Wanted Phrase", wantedPhrase)
		assert.NotNil(t, err)
		assert.ErrorContains(t, errIncorrectPhrase.Error(), err)
	})
}
