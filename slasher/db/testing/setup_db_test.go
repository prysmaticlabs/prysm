package testing

import (
	"flag"
	"os"
	"testing"

	"gopkg.in/urfave/cli.v2"
)

func TestClearDB(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	slasherDB := SetupSlasherDB(t, cli.NewContext(&app, set, nil))
	defer TeardownSlasherDB(t, slasherDB)
	if err := slasherDB.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(slasherDB.DatabasePath()); !os.IsNotExist(err) {
		t.Fatalf("db wasnt cleared %v", err)
	}
}
