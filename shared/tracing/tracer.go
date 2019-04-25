package tracing

import (
	"errors"

	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "tracing")

// Setup creates and initializes a new tracing configuration..
func Setup(name, endpoint string, sampleFraction float64, enable bool) error {
	if !enable {
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.NeverSample()})
		return nil
	}

	if name == "" {
		return errors.New("tracing service name cannot be empty")
	}

	trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(sampleFraction)})

	log.Infof("Starting Jaeger exporter endpoint at address = %s", endpoint)
	exporter, err := jaeger.NewExporter(jaeger.Options{
		Endpoint: endpoint,
		Process: jaeger.Process{
			ServiceName: name,
		},
	})
	if err != nil {
		return err
	}
	trace.RegisterExporter(exporter)

	return nil
}
