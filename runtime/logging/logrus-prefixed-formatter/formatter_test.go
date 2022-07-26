package prefixed_test

import (
	"testing"

	"github.com/pkg/errors"
	. "github.com/prysmaticlabs/prysm/runtime/logging/logrus-prefixed-formatter"
	"github.com/prysmaticlabs/prysm/testing/require"

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

func TestFormatter_SuppressErrorStackTraces(t *testing.T) {
	formatter := new(TextFormatter)
	formatter.ForceFormatting = true
	log := logrus.New()
	log.Formatter = formatter
	output := new(LogOutput)
	log.Out = output

	errFn := func() error {
		return errors.New("inner")
	}

	log.WithError(errors.Wrap(errFn(), "outer")).Error("test")
	require.Equal(t, "[0000] ERROR test error=inner", output.GetValue(), "wrong log output")
}
