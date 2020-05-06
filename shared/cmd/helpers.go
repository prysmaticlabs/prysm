package cmd

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

var log = logrus.WithField("prefix", "node")

// ConfirmAction uses the passed in actionText as the confirmation text displayed in the terminal.
// The user must enter Y or N to indicate whether they confirm the action detailed in the warning text.
// Returns a boolean representing the user's answer.
func ConfirmAction(actionText string, deniedText string) (bool, error) {
	var confirmed bool
	reader := bufio.NewReader(os.Stdin)
	log.Warn(actionText)

	for {
		fmt.Print(">> ")

		line, _, err := reader.ReadLine()
		if err != nil {
			return false, err
		}
		trimmedLine := strings.TrimSpace(string(line))
		lineInput := strings.ToUpper(trimmedLine)
		if lineInput != "Y" && lineInput != "N" {
			log.Errorf("Invalid option of %s chosen, please only enter Y/N", line)
			continue
		}
		if lineInput == "Y" {
			confirmed = true
			break
		}
		log.Warn(deniedText)
		break
	}

	return confirmed, nil
}

// EnterPassword queries the user for their password through the terminal, in order to make sure it is
// not passed in a visible way to the terminal.
// TODO(#5749): This function is untested and should be tested.
func EnterPassword() (string, error) {
	var passphrase string
	log.Info("Enter a password:")
	bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return passphrase, errors.Wrap(err, "could not read account password")
	}
	text := string(bytePassword)
	passphrase = strings.Replace(text, "\n", "", -1)
	return passphrase, nil
}
