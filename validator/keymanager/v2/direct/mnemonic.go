package direct

import (
	"fmt"

	"github.com/brianium/mnemonic"
	"github.com/brianium/mnemonic/entropy"
	"github.com/manifoldco/promptui"
)

const (
	mnemonicLanguage = mnemonic.English
)

// MnemonicFactory defines a struct which
// can generate new seed phrases in human-readable
// format from a source of entropy in raw bytes. It
// also provides methods for verifying a user has successfully
// acknowledged the mnemonic phrase and written it down offline.
type SeedPhraseFactory interface {
	Generate(data []byte) (*mnemonic.Mnemonic, error)
	ConfirmAcknowledgement(phrase *mnemonic.Mnemonic) error
}

// EnglishMnemonicGenerator implements methods for creating
// mnemonic seed phrases in english using a given
// source of entropy such as a private key.
type EnglishMnemonicGenerator struct{}

// Generate a mnemonic seed phrase in english using a source of
// entropy given as raw bytes.
func (m *EnglishMnemonicGenerator) Generate(data []byte) (*mnemonic.Mnemonic, error) {
	ent, err := entropy.FromHex(fmt.Sprintf("%x", data))
	if err != nil {
		return nil, err
	}
	return mnemonic.New(ent, mnemonicLanguage)
}

// ConfirmAcknowledgement displays the mnemonic phrase to the user
// and confirms the user has written down the phrase securely offline.
func (m *EnglishMnemonicGenerator) ConfirmAcknowledgement(mnemonic *mnemonic.Mnemonic) error {
	log.Info(
		"Write down the sentence below, as it is your only " +
			"means of recovering your withdrawal key",
	)
	fmt.Printf(`
=================Withdrawal Key Recovery Phrase====================

%s

===================================================================
	`, mnemonic.Sentence())
	// Confirm the user has written down the mnemonic phrase offline.
	prompt := promptui.Prompt{
		Label:     "Confirm you have written down the recovery words somewhere safe (offline)",
		IsConfirm: true,
	}
	expected := "y"
	var result string
	var err error
	for result != expected {
		result, err = prompt.Run()
		if err != nil {
			log.Errorf("Could not confirm acknowledgement of prompt, please enter y")
		}
	}
	return nil
}
