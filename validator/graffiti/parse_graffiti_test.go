package graffiti

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestParseGraffitiFile_Default(t *testing.T) {
	input := []byte(`default: "Mr T was here"`)

	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	someFileName := filepath.Join(dirName, "somefile.txt")
	require.NoError(t, ioutil.WriteFile(someFileName, input, os.ModePerm))

	got, err := ParseGraffitiFile(someFileName)
	require.NoError(t, err)

	wanted := &Graffiti{
		DefaultGraffiti: "Mr T was here",
	}
	require.DeepEqual(t, wanted, got)
}

func TestParseGraffitiFile_Random(t *testing.T) {
	input := []byte(`random: 
  - "Mr A was here"
  - "Mr B was here"
  - "Mr C was here"`)

	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	someFileName := filepath.Join(dirName, "somefile.txt")
	require.NoError(t, ioutil.WriteFile(someFileName, input, os.ModePerm))

	got, err := ParseGraffitiFile(someFileName)
	require.NoError(t, err)

	wanted := &Graffiti{
		RandomGraffiti: []string{
			"Mr A was here",
			"Mr B was here",
			"Mr C was here",
		},
	}
	require.DeepEqual(t, wanted, got)
}

func TestParseGraffitiFile_Validators(t *testing.T) {
	input := []byte(`
validators: 
  1234: Yolo
  555: "What's up"
  703727: Meow`)

	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	someFileName := filepath.Join(dirName, "somefile.txt")
	require.NoError(t, ioutil.WriteFile(someFileName, input, os.ModePerm))

	got, err := ParseGraffitiFile(someFileName)
	require.NoError(t, err)

	wanted := &Graffiti{
		ValidatorGraffiti: map[uint64]string{
			1234:   "Yolo",
			555:    "What's up",
			703727: "Meow",
		},
	}
	require.DeepEqual(t, wanted, got)
}

func TestParseGraffitiFile_AllFields(t *testing.T) {
	input := []byte(`default: "Mr T was here"
random: 
  - "Mr A was here"
  - "Mr B was here"
  - "Mr C was here"

validators: 
  1234: Yolo
  555: "What's up"
  703727: Meow`)

	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	someFileName := filepath.Join(dirName, "somefile.txt")
	require.NoError(t, ioutil.WriteFile(someFileName, input, os.ModePerm))

	got, err := ParseGraffitiFile(someFileName)
	require.NoError(t, err)

	wanted := &Graffiti{
		DefaultGraffiti: "Mr T was here",
		RandomGraffiti: []string{
			"Mr A was here",
			"Mr B was here",
			"Mr C was here",
		},
		ValidatorGraffiti: map[uint64]string{
			1234:   "Yolo",
			555:    "What's up",
			703727: "Meow",
		},
	}
	require.DeepEqual(t, wanted, got)
}
