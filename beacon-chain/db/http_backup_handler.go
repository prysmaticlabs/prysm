package db

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// BackupHandler for accepting requests to initiate a new database backup.
func BackupHandler(db Database) func(http.ResponseWriter, *http.Request) {
	log := logrus.WithField("prefix", "db")

	return func(w http.ResponseWriter, _ *http.Request) {
		log.Debug("Creating database backup from HTTP webhook.")

		if err := db.Backup(context.Background()); err != nil {
			log.WithError(err).Error("Failed to create backup")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, "OK")
		if err != nil {
			log.WithError(err).Error("Failed to write OK")
		}
	}
}
