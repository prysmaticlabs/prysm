package derived

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/promptutil"

	"github.com/tyler-smith/go-bip39"
)

const confirmationText = "Confirm you have written down the recovery words somewhere safe (offline) [y|Y]"

// SeedPhraseFactory defines a struct which
// can generate new seed phrases in human-readable
// format from a source of entropy in raw bytes. It
// also provides methods for verifying a user has successfully
// acknowledged the mnemonic phrase and written it down offline.
type SeedPhraseFactory interface {
	Generate(data []byte) (string, error)
	ConfirmAcknowledgement(phrase string) error
}

// EnglishMnemonicGenerator implements methods for creating
// mnemonic seed phrases in english using a given
// source of entropy such as a private key.
type EnglishMnemonicGenerator struct {
	skipMnemonicConfirm bool
}

// Generate a mnemonic seed phrase in english using a source of
// entropy given as raw bytes.
func (m *EnglishMnemonicGenerator) Generate(data []byte) (string, error) {
	return bip39.NewMnemonic(data)
}

// ConfirmAcknowledgement displays the mnemonic phrase to the user
// and confirms the user has written down the phrase securely offline.
func (m *EnglishMnemonicGenerator) ConfirmAcknowledgement(phrase string) error {
	log.Info(
		"Write down the sentence below, as it is your only " +
			"means of recovering your wallet",
	)
	fmt.Printf(
		`=================Wallet Seed Recovery Phrase====================

%s

===================================================================`,
		phrase)
	fmt.Println("")
	if m.skipMnemonicConfirm {
		return nil
	}
	// Confirm the user has written down the mnemonic phrase offline.
	_, err := promptutil.ValidatePrompt(confirmationText, promptutil.ValidateConfirmation)
	if err != nil {
		log.Errorf("Could not confirm acknowledgement of prompt, please enter y")
	}
	return nil
}
