package main

import (
	"time"
)

var keyInterval = 3 * time.Minute

type keyChecker struct {
	db *db
}

func newkeyChecker(db *db, beaconRPCAddr string) *keyChecker {
	log.Warn("Key checker temporarily disabled during refactor.")

	return &keyChecker{
		db: db,
	}
}

func (k *keyChecker) checkKeys() error {
	log.Warn("Not checking for EXITED validator keys.")
	return nil
}

func (k *keyChecker) run() {
	for {
		if err := k.checkKeys(); err != nil {
			log.WithError(err).Error("Failed to check keys")
		}
		time.Sleep(keyInterval)
	}
}
