// +build !develop

package params

import (
	log "github.com/sirupsen/logrus"
)

func init() {
	log.Fatal("Tests in this package require extra build tag: re-run with `-tags develop`")
}
