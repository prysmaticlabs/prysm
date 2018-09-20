package tracer

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "tracer")

// New creates and initializes a new tracing adapter.
func New(name, endpoint string, sampleFraction float64, enable bool) (p2p.Adapter, error) {
	if !enable {
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.NeverSample()})
		return adapter, nil
	}

	if name == "" {
		return nil, errors.New("tracing service name cannot be empty")
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
		return nil, err
	}
	trace.RegisterExporter(exporter)

	return adapter, nil
}

var adapter p2p.Adapter = func(next p2p.Handler) p2p.Handler {
	return func(msg p2p.Message) {
		var messageSpan *trace.Span
		msg.Ctx, messageSpan = trace.StartSpan(msg.Ctx, "handleP2pMessage")
		next(msg)
		messageSpan.End()
	}
}
