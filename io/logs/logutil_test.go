package logs

import (
	"fmt"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

var urltests = []struct {
	url       string
	maskedUrl string
}{
	{"https://a:b@xyz.net", "https://***@xyz.net"},
	{"https://eth-goerli.alchemyapi.io/v2/tOZG5mjl3.zl_nZdZTNIBUzsDq62R_dkOtY",
		"https://eth-goerli.alchemyapi.io/***"},
	{"https://google.com/search?q=golang", "https://google.com/***"},
	{"https://user@example.com/foo%2fbar", "https://***@example.com/***"},
	{"http://john@example.com/#x/y%2Fz", "http://***@example.com/#***"},
	{"https://me:pass@example.com/foo/bar?x=1&y=2", "https://***@example.com/***"},
}

func TestMaskCredentialsLogging(t *testing.T) {
	for _, test := range urltests {
		require.Equal(t, MaskCredentialsLogging(test.url), test.maskedUrl)
	}
}

func TestConfigurePersistantLogging(t *testing.T) {
	// 1. Test creation of file in an existing parent directory
	logFileName := "test.log"
	existingDirectory := "existing-testing-dir"
	err := os.Mkdir(existingDirectory, 0700)
	if err != nil {
		return
	}

	err = ConfigurePersistentLogging(fmt.Sprintf("%s/%s/%s", t.TempDir(), existingDirectory, logFileName))
	require.NoError(t, err)

	err = os.RemoveAll(existingDirectory)
	if err != nil {
		return
	}

	// 2. Test creation of file along with parent directory
	nonExistingDirectory := "non-existing-testing-dir"

	err = ConfigurePersistentLogging(fmt.Sprintf("%s/%s/%s", t.TempDir(), nonExistingDirectory, logFileName))
	require.NoError(t, err)

	err = os.RemoveAll(nonExistingDirectory)
	if err != nil {
		return
	}

	// 3. Test creation of file in an existing parent directory with a non-existing sub-directory
	existingDirectory = "existing-testing-dir"
	nonExistingSubDirectory := "non-existing-sub-dir"
	err = os.Mkdir(existingDirectory, 0700)
	if err != nil {
		return
	}

	err = ConfigurePersistentLogging(fmt.Sprintf("%s/%s/%s/%s", t.TempDir(), existingDirectory, nonExistingSubDirectory, logFileName))
	require.NoError(t, err)

	err = os.RemoveAll(existingDirectory)
	if err != nil {
		return
	}

	// 4. Create log file in a directory without 700 permissions
	existingDirectory = "existing-testing-dir"
	err = os.Mkdir(existingDirectory, 0750)
	if err != nil {
		return
	}

	err = ConfigurePersistentLogging(fmt.Sprintf("%s/%s/%s", t.TempDir(), existingDirectory, logFileName))
	require.ErrorIs(t, err, errors.New("dir already exists without proper 0700 permissions"))
}
