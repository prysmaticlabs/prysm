package tracer

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

const (
	defaultSamplingFraction = 0.25
)

var log = logrus.WithField("prefix", "tracer")

func New(name, endpoint string, disable bool) (p2p.Adapter, error) {
	if disable {
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.NeverSample()})
		return adapter, nil
	}

	if name == "" {
		return nil, errors.New("tracing service name cannot be empty")
	}

	// TODO: make sampling fraction configurable?
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(defaultSamplingFraction)})

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
	return func(ctx context.Context, msg p2p.Message) {
		ctx, messageSpan := trace.StartSpan(ctx, "handleP2pMessage")
		msg.Ctx = ctx
		next(ctx, msg)
		messageSpan.End()
	}
}
