package kv

import (
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "db")

func logWhileFuncRun(intialMessage string, progressMessage string, delay time.Duration) chan string {
	stopMs := make(chan string)
	log.Debugln(intialMessage)
	go func() {
		for {
			log.Debugln(progressMessage)
			select {
			case <-time.After(delay):
			case ms := <-stopMs:
				log.Debugln(ms)
				return
			}
		}
	}()
	return stopMs
}
