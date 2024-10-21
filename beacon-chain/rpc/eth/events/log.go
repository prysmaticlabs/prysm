package events

import "github.com/sirupsen/logrus"

var logger = logrus.New()
var log = logger.WithField("prefix", "events")
