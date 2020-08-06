package fuzz

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestBeaconFuzzBlock(t *testing.T) {
	defer func() {
		if err := os.RemoveAll(dbPath); err != nil {
			panic(err)
		}
	}()
	dir, ok := os.LookupEnv("CORPUS")
	if !ok {
		t.Fatal("No environment variable set for CORPUS")
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		p := path.Join(dir, f.Name())
		b, err := ioutil.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		//t.Run(f.Name(), func(t *testing.T) {
		BeaconFuzzBlock(b)
		//})
	}
}
