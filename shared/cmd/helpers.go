package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "node")

// ConfirmAction uses the passed in actionText as the confirmation text displayed in the terminal.
// The user must enter Y or N to indicate whether they confirm the action detailed in the warning text.
// Returns a boolean representing the user's answer.
func ConfirmAction(actionText, deniedText string) (bool, error) {
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
func EnterPassword(confirmPassword bool, pr PasswordReader) (string, error) {
	var passphrase string
	log.Info("Enter a password:")
	bytePassword, err := pr.ReadPassword()
	if err != nil {
		return "", errors.Wrap(err, "could not read account password")
	}
	text := bytePassword
	passphrase = strings.ReplaceAll(text, "\n", "")
	if confirmPassword {
		log.Info("Please re-enter your password:")
		bytePassword, err := pr.ReadPassword()
		if err != nil {
			return "", errors.Wrap(err, "could not read account password")
		}
		text := bytePassword
		confirmedPass := strings.ReplaceAll(text, "\n", "")
		if passphrase != confirmedPass {
			log.Info("Passwords did not match, please try again")
			return EnterPassword(true, pr)
		}
	}
	return passphrase, nil
}

// expandWeb3EndpointIfFile expends the path for --http-web3provider if specified as a file
func ExpandWeb3EndpointIfFile(ctx *cli.Context, flag *cli.StringFlag) error {
	web3endpoint := ctx.String(flag.Name)
	if !strings.HasPrefix(web3endpoint, "http://") &&
		!strings.HasPrefix(web3endpoint, "https://") &&
		!strings.HasPrefix(web3endpoint, "ws://") &&
		!strings.HasPrefix(web3endpoint, "wss://") {
		web3endpoint, err := fileutil.ExpandPath(ctx.String(flag.Name))
		if err != nil {
			return errors.Wrapf(err, "could not expand path for %s", web3endpoint)
		}
		if err := ctx.Set(flag.Name, web3endpoint); err != nil {
			return errors.Wrapf(err, "could not set %s to %s", flag.Name, web3endpoint)
		}
	}
	return nil
}
