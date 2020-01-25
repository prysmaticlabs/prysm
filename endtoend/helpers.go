package endtoend

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func killProcesses(t *testing.T, pIDs []int) {
	for _, id := range pIDs {
		process, err := os.FindProcess(id)
		if err != nil {
			t.Fatalf("Could not find process %d: %v", id, err)
		}
		if err := process.Kill(); err != nil {
			t.Fatal(err)
		}
		if err := process.Release(); err != nil {
			t.Fatal(err)
		}
	}
}

func waitForTextInFile(file *os.File, text string) error {
	wait := 0
	// Cap the wait in case there are issues starting.
	maxWait := 36
	for wait < maxWait {
		time.Sleep(2 * time.Second)
		// Rewind the file pointer to the start of the file so we can read it again.
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, "could not rewind file to start")
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), text) {
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		wait += 2
	}
	contents, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return err
	}
	return fmt.Errorf("could not find requested text \"%s\" in logs:\n%s", text, string(contents))
}

func logOutput(t *testing.T, tmpPath string, config *end2EndConfig) {
	// Log out errors from beacon chain nodes.
	for i := uint64(0); i < config.numBeaconNodes; i++ {
		beaconLogFile, err := os.Open(path.Join(tmpPath, fmt.Sprintf(beaconNodeLogFileName, i)))
		if err != nil {
			t.Fatal(err)
		}
		logErrorOutput(t, beaconLogFile, "beacon chain node", i)

		validatorLogFile, err := os.Open(path.Join(tmpPath, fmt.Sprintf(validatorLogFileName, i)))
		if err != nil {
			t.Fatal(err)
		}
		logErrorOutput(t, validatorLogFile, "validator client", i)
	}
	t.Logf("Ending time: %s\n", time.Now().String())
}

func logErrorOutput(t *testing.T, file *os.File, title string, index uint64) {
	var errorLines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		currentLine := scanner.Text()
		//if strings.Contains(currentLine, "level=error") {
		errorLines = append(errorLines, currentLine)
		//}
	}

	if len(errorLines) < 1 {
		t.Logf("No error logs detected for %s %d", title, index)
		return
	}

	t.Log("===================================================================")
	t.Logf("Start of %s %d error output:\n", title, index)

	for _, err := range errorLines {
		t.Log(err)
	}

	t.Logf("\nEnd of %s %d error output:", title, index)
	t.Log("===================================================================")
}
