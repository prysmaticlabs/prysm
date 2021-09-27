package slasher

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "slasher")

type Service struct {
	params *Parameters
}
