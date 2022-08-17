package derived

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/crypto/rand"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	"github.com/tyler-smith/go-bip39"
)

const confirmationText = "Confirm you have written down the recovery words somewhere safe (offline) [y|Y]"

// EnglishMnemonicGenerator implements methods for creating
// mnemonic seed phrases in english using a given
// source of entropy such as a private key.
type EnglishMnemonicGenerator struct {
	skipMnemonicConfirm bool
}

// GenerateAndConfirmMnemonic requires confirming the generated mnemonics.
func GenerateAndConfirmMnemonic(
	skipMnemonicConfirm bool,
) (string, error) {
	mnemonicRandomness := make([]byte, 32)
	if _, err := rand.NewGenerator().Read(mnemonicRandomness); err != nil {
		return "", errors.Wrap(err, "could not initialize mnemonic source of randomness")
	}
	m := &EnglishMnemonicGenerator{
		skipMnemonicConfirm: skipMnemonicConfirm,
	}
	phrase, err := m.Generate(mnemonicRandomness)
	if err != nil {
		return "", errors.Wrap(err, "could not generate wallet seed")
	}
	if err := m.ConfirmAcknowledgement(phrase); err != nil {
		return "", errors.Wrap(err, "could not confirm mnemonic acknowledgement")
	}
	return phrase, nil
}

// Generate a mnemonic seed phrase in english using a source of
// entropy given as raw bytes.
func (_ *EnglishMnemonicGenerator) Generate(data []byte) (string, error) {
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
	_, err := prompt.ValidatePrompt(os.Stdin, confirmationText, prompt.ValidateConfirmation)
	if err != nil {
		log.Errorf("Could not confirm acknowledgement of userprompt, please enter y")
	}
	return nil
}

// Uses the provided mnemonic seed phrase to generate the
// appropriate seed file for recovering a derived wallets.
func seedFromMnemonic(mnemonic, mnemonicPassphrase string) ([]byte, error) {
	if ok := bip39.IsMnemonicValid(mnemonic); !ok {
		return nil, bip39.ErrInvalidMnemonic
	}
	return bip39.NewSeed(mnemonic, mnemonicPassphrase), nil
}
