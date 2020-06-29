package v2

import (
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	logrus "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts-v2")

// Steps: ask for the path to store the validator datadir
// Ask for the type of wallet: direct, derived, remote
// Ask for where to store passwords: default path within datadir
// Ask them to enter a password for the account.
// If user already has account, instead just make a new account with a good name
// Allow for creating more than 1 validator??
// TODOS: mnemonic for withdrawal key, ensure they write it down.
func New(cliCtx *cli.Context) error {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("wallet directory path must not be empty")
		}
		return nil
	}
	datadir := cliCtx.String(cmd.DataDirFlag.Name)
	if datadir == "" {
		// Maybe too aggressive...
		log.Fatal("Could not determine your system's home path")
	}

	prompt := promptui.Prompt{
		Label:    "Enter a wallet directory",
		Validate: validate,
		Default:  datadir,
	}
	result, err := prompt.Run()
	if err != nil {
		switch err {
		case promptui.ErrAbort:
			log.Fatal("Wallet creation aborted, closing...")
		case promptui.ErrInterrupt:
			log.Fatal("Keyboard interrupt, closing...")
		case promptui.ErrEOF:
			log.Fatal("No input received, closing...")
		default:
			log.Fatalf("Could not complete wallet creation: %v", err)
		}
	}

	promptSelect := promptui.Select{
		Label: "Select a type of wallet",
		Items: []string{
			"Direct, On-Disk (Recommended)",
			"Derived (Advanced)",
			"Remote (Advanced)",
		},
	}

	_, resultSelect, err := promptSelect.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil
	}
	fmt.Println(resultSelect)
	validate = func(input string) error {
		if len(input) < 6 {
			return errors.New("Password must have more than 6 characters")
		}
		return nil
	}

	prompt = promptui.Prompt{
		Label:    "Password",
		Validate: validate,
		Mask:     '*',
	}

	result, err = prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil
	}

	fmt.Printf("Your password is %q\n", result)
	prompt = promptui.Prompt{
		Label:     "Delete Resource",
		IsConfirm: true,
	}

	result, err = prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil
	}

	fmt.Printf("You choose %q\n", result)
	return nil
}
