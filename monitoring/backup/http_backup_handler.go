package backup

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// BackupExporter defines a backup exporter methods.
type BackupExporter interface {
	Backup(ctx context.Context, outputPath string, permissionOverride bool) error
}

// BackupHandler for accepting requests to initiate a new database backup.
func BackupHandler(bk BackupExporter, outputDir string) func(http.ResponseWriter, *http.Request) {
	log := logrus.WithField("prefix", "db")

	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Creating database backup from HTTP webhook")

		_, permissionOverride := r.URL.Query()["permissionOverride"]

		if err := bk.Backup(context.Background(), outputDir, permissionOverride); err != nil {
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
