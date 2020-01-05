package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
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
