package endtoend

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

const (
	maxPollingWaitTime = 36 * time.Second
	filePollingInterval = 1 * time.Second
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

func waitForTextInFile(file io.Reader, text string) error {
	d := time.Now().Add(maxPollingWaitTime)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	// Use a ticker with a deadline to poll a given file.
	ticker := time.NewTicker(filePollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			contents, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}
			return fmt.Errorf("could not find requested text \"%s\" in logs:\n%s", text, string(contents))
		case <-ticker.C:
			fmt.Println("ddd")
			// Checking the file for contents...
			enc, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}
			fmt.Println(enc)
			if strings.Contains(string(enc), text) {
				return nil
			}
		}
	}
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

func logErrorOutput(t *testing.T, file io.Reader, title string, index uint64) {
	var errorLines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		currentLine := scanner.Text()
		if strings.Contains(currentLine, "level=error") {
			errorLines = append(errorLines, currentLine)
		}
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
