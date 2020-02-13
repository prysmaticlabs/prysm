package testing

import (
	"flag"
	"os"
	"testing"

	"github.com/urfave/cli"
)

func TestClearDB(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	slasherDB := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, slasherDB)
	if err := slasherDB.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(slasherDB.DatabasePath()); !os.IsNotExist(err) {
		t.Fatalf("db wasnt cleared %v", err)
	}
}
