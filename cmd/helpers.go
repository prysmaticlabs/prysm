package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/io/file"
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

// ExpandSingleEndpointIfFile expands the path for --http-web3provider if specified as a file.
func ExpandSingleEndpointIfFile(ctx *cli.Context, flag *cli.StringFlag) error {
	// Return early if no flag value is set.
	if !ctx.IsSet(flag.Name) {
		return nil
	}
	web3endpoint := ctx.String(flag.Name)
	switch {
	case strings.HasPrefix(web3endpoint, "http://"):
	case strings.HasPrefix(web3endpoint, "https://"):
	case strings.HasPrefix(web3endpoint, "ws://"):
	case strings.HasPrefix(web3endpoint, "wss://"):
	default:
		web3endpoint, err := file.ExpandPath(ctx.String(flag.Name))
		if err != nil {
			return errors.Wrapf(err, "could not expand path for %s", web3endpoint)
		}
		if err := ctx.Set(flag.Name, web3endpoint); err != nil {
			return errors.Wrapf(err, "could not set %s to %s", flag.Name, web3endpoint)
		}
	}
	return nil
}

// ExpandWeb3EndpointsIfFile expands the path for --fallback-web3provider if specified as a file.
func ExpandWeb3EndpointsIfFile(ctx *cli.Context, flags *cli.StringSliceFlag) error {
	// Return early if no flag value is set.
	if !ctx.IsSet(flags.Name) {
		return nil
	}
	rawFlags := ctx.StringSlice(flags.Name)
	for i, rawValue := range rawFlags {
		switch {
		case strings.HasPrefix(rawValue, "http://"):
		case strings.HasPrefix(rawValue, "https://"):
		case strings.HasPrefix(rawValue, "ws://"):
		case strings.HasPrefix(rawValue, "wss://"):
		default:
			web3endpoint, err := file.ExpandPath(rawValue)
			if err != nil {
				return errors.Wrapf(err, "could not expand path for %s", rawValue)
			}
			// Given that rawFlags is a pointer this will replace the unexpanded path
			// with the expanded one. Also there is no easy way to replace the string
			// slice flag value compared to other flag types. This is why we resort to
			// replacing it like this.
			rawFlags[i] = web3endpoint
		}
	}
	return nil
}
