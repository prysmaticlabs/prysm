package derived

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/tyler-smith/go-bip39"
	"github.com/tyler-smith/go-bip39/wordlists"
)

func TestMnemonic_Generate_CanRecover(t *testing.T) {
	generator := &MnemonicGenerator{}
	data := make([]byte, 32)
	copy(data, "hello-world")
	phrase, err := generator.Generate(data)
	require.NoError(t, err)
	entropy, err := bip39.EntropyFromMnemonic(phrase)
	require.NoError(t, err)
	assert.DeepEqual(t, data, entropy, "Expected to recover original data")
}

func Test_setBip39Lang(t *testing.T) {
	tests := []struct {
		lang             string
		expectedWordlist []string
	}{
		{lang: "english", expectedWordlist: wordlists.English},
		{lang: "chinese_traditional", expectedWordlist: wordlists.ChineseTraditional},
		{lang: "chinese_simplified", expectedWordlist: wordlists.ChineseSimplified},
		{lang: "czech", expectedWordlist: wordlists.Czech},
		{lang: "french", expectedWordlist: wordlists.French},
		{lang: "japanese", expectedWordlist: wordlists.Japanese},
		{lang: "korean", expectedWordlist: wordlists.Korean},
		{lang: "italian", expectedWordlist: wordlists.Italian},
		{lang: "spanish", expectedWordlist: wordlists.Spanish},
		{lang: "undefined", expectedWordlist: wordlists.English},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			setBip39Lang(tt.lang)
			wordlist := bip39.GetWordList()
			assert.DeepEqual(t, tt.expectedWordlist, wordlist, "Expected wordlist to match")
		})
	}
}
