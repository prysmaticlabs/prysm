package node

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

func confirmDelete(d db.Database, path string) (db.Database, error) {
	var clearDB bool
	var err error
	reader := bufio.NewReader(os.Stdin)

	log.Warn("This will delete your beacon chain data base stored in your data directory. " +
		"Your database backups will not be removed - do you want to proceed? (Y/N)")

	for {
		fmt.Print(">> ")

		line, _, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		trimmedLine := strings.TrimSpace(string(line))
		lineInput := strings.ToUpper(trimmedLine)
		if lineInput != "Y" && lineInput != "N" {
			log.Errorf("Invalid option of %s chosen, enter Y/N", line)
			continue
		}
		if lineInput == "Y" {
			log.Warn("Deleting beaconchain.db from data directory")
			clearDB = true
			break
		}
		log.Info("Not deleting chain database, the db will be initialized" +
			" with the current data directory.")
		break
	}

	if clearDB {
		if err := d.ClearDB(); err != nil {
			return nil, err
		}
		d, err = db.NewDB(path)
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}
