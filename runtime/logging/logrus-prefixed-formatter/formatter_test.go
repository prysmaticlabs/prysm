package prefixed_test

import (
	. "github.com/prysmaticlabs/prysm/runtime/logging/logrus-prefixed-formatter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Formatter", func() {
	var formatter *TextFormatter
	var log *logrus.Logger
	var output *LogOutput

	BeforeEach(func() {
		output = new(LogOutput)
		formatter = new(TextFormatter)
		log = logrus.New()
		log.Out = output
		log.Formatter = formatter
		log.Level = logrus.DebugLevel
	})

	Describe("logfmt output", func() {
		It("should output simple message", func() {
			formatter.DisableTimestamp = true
			log.Debug("test")
			Ω(output.GetValue()).Should(Equal("level=debug msg=test\n"))
		})

		It("should output message with additional field", func() {
			formatter.DisableTimestamp = true
			log.WithFields(logrus.Fields{"animal": "walrus"}).Debug("test")
			Ω(output.GetValue()).Should(Equal("level=debug msg=test animal=walrus\n"))
		})
	})

	Describe("Formatted output", func() {
		It("should output formatted message", func() {
			formatter.DisableTimestamp = true
			formatter.ForceFormatting = true
			log.Debug("test")
			Ω(output.GetValue()).Should(Equal("DEBUG test\n"))
		})
	})

	Describe("Theming support", func() {

	})
})
